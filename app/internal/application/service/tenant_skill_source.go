package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	apperrors "gaze/internal/errors"
	secutils "gaze/internal/utils"
)

// ErrSkillSourceInvalid marks every rejection of a git / URL source so the
// handler can map the class to 400 without matching on text.
var ErrSkillSourceInvalid = errors.New("skill source is invalid")

const (
	skillSourceUserAgent    = "Gaze-SkillInstaller (+https://gaze)"
	skillSourceFetchTimeout = 5 * time.Minute
)

var (
	skillSourceHTTPOnce    sync.Once
	skillSourceHTTPDefault *http.Client
)

type skillSourceKind string

const (
	skillSourceGitHub skillSourceKind = "github"
	skillSourceGitLab skillSourceKind = "gitlab"
	skillSourceDirect skillSourceKind = "direct"
)

// parsedSkillSource is one install input after host-specific URL rules have
// been applied, and before any bytes are fetched.
type parsedSkillSource struct {
	Kind      skillSourceKind
	Owner     string
	Repo      string
	Ref       string
	Subdir    string
	DirectURL string
}

// InstallSkillFromSource resolves a git / direct-zip locator to a skill
// bundle and runs the same install as an upload.
//
// Every fetch is anonymous: no credential may be presented to any host.
func (s *TenantSkillService) InstallSkillFromSource(
	ctx context.Context, tenantID uint64, configID, source string,
) (string, error) {
	// The config is authorized before the fetch, not by InstallSkill after it.
	// The source is a caller-supplied host, so an unknown config ID must not
	// be able to spend an outbound request and a body-sized download first.
	cfgEntity, err := s.configs.GetByID(ctx, tenantID, configID)
	if err != nil {
		return "", err
	}
	if cfgEntity == nil {
		return "", apperrors.NewNotFoundError("sandbox config not found")
	}

	bundle, archive, err := fetchNormalizedSkillBundle(ctx, source, s.sourceHTTP)
	if err != nil {
		return "", err
	}
	return s.installParsedSkill(ctx, tenantID, configID, bundle, archive)
}

func skillSourceHTTPClient(override *http.Client) *http.Client {
	if override != nil {
		return override
	}
	skillSourceHTTPOnce.Do(func() {
		cfg := secutils.DefaultSSRFSafeHTTPClientConfig()
		cfg.Timeout = skillSourceFetchTimeout
		skillSourceHTTPDefault = secutils.NewSSRFSafeHTTPClient(cfg)
	})
	return skillSourceHTTPDefault
}

func fetchSkillArchive(ctx context.Context, source string, client *http.Client) ([]byte, error) {
	_, archive, err := fetchNormalizedSkillBundle(ctx, source, client)
	return archive, err
}

func fetchNormalizedSkillBundle(
	ctx context.Context, source string, client *http.Client,
) (*SkillBundle, []byte, error) {
	parsed, err := parseSkillSource(source)
	if err != nil {
		return nil, nil, err
	}
	httpClient := skillSourceHTTPClient(client)
	fetched, err := fetchSkillSourceBytes(ctx, httpClient, parsed)
	if err != nil {
		return nil, nil, err
	}
	return normalizeFetchedSkill(fetched.body, fetched.contentType, fetched.subdir)
}

// parseSkillSource maps one paste onto exactly one kind: a github.com /
// gitlab.com URL, or a direct zip/SKILL.md URL. Bare slugs are refused rather
// than guessed.
func parseSkillSource(raw string) (parsedSkillSource, error) {
	input := strings.TrimSpace(raw)
	if input == "" {
		return parsedSkillSource{}, fmt.Errorf("%w: source is required", ErrSkillSourceInvalid)
	}
	input = strings.TrimRight(input, "/")

	if strings.Contains(input, "://") {
		return parseSkillSourceURL(input)
	}
	return parsedSkillSource{}, fmt.Errorf(
		"%w: %q is not a URL; paste a github.com / gitlab.com URL or a direct zip / SKILL.md link",
		ErrSkillSourceInvalid, input)
}

func parseSkillSourceURL(raw string) (parsedSkillSource, error) {
	parsedURL, err := url.Parse(raw)
	if err != nil {
		return parsedSkillSource{}, fmt.Errorf("%w: not a valid URL", ErrSkillSourceInvalid)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return parsedSkillSource{}, fmt.Errorf("%w: only http(s) sources are allowed", ErrSkillSourceInvalid)
	}
	host := strings.ToLower(parsedURL.Hostname())
	switch {
	case host == "github.com" || host == "www.github.com" || host == "codeload.github.com":
		if host == "codeload.github.com" {
			return parsedSkillSource{Kind: skillSourceDirect, DirectURL: parsedURL.String()}, nil
		}
		return parseGitHubURL(parsedURL)
	case host == "gitlab.com" || host == "www.gitlab.com":
		return parseGitLabURL(parsedURL)
	default:
		if isDirectArchivePath(parsedURL.Path) {
			return parsedSkillSource{Kind: skillSourceDirect, DirectURL: parsedURL.String()}, nil
		}
		return parsedSkillSource{}, fmt.Errorf(
			"%w: unsupported host; paste a github.com / gitlab.com URL or a direct zip / SKILL.md link",
			ErrSkillSourceInvalid)
	}
}

func parseGitHubURL(u *url.URL) (parsedSkillSource, error) {
	parts := splitPath(u.Path)
	if len(parts) < 2 {
		return parsedSkillSource{}, fmt.Errorf("%w: github URL must be owner/repo", ErrSkillSourceInvalid)
	}
	owner, repo := parts[0], strings.TrimSuffix(parts[1], ".git")
	src := parsedSkillSource{Kind: skillSourceGitHub, Owner: owner, Repo: repo, Ref: "HEAD"}
	rest := parts[2:]
	if len(rest) == 0 {
		return src, nil
	}
	switch rest[0] {
	case "tree", "blob":
		if len(rest) < 2 {
			return parsedSkillSource{}, fmt.Errorf("%w: github tree URL is missing a ref", ErrSkillSourceInvalid)
		}
		src.Ref = rest[1]
		src.Subdir = strings.Join(rest[2:], "/")
		if rest[0] == "blob" && strings.EqualFold(path.Base(src.Subdir), "SKILL.md") {
			src.Subdir = path.Dir(src.Subdir)
			if src.Subdir == "." {
				src.Subdir = ""
			}
		}
	case "archive":
		src.DirectURL = u.String()
		src.Kind = skillSourceDirect
	case "releases":
		src.DirectURL = u.String()
		src.Kind = skillSourceDirect
	default:
		src.Subdir = strings.Join(rest, "/")
	}
	return src, nil
}

func parseGitLabURL(u *url.URL) (parsedSkillSource, error) {
	trimmed := strings.Trim(u.Path, "/")
	project, extra, found := strings.Cut(trimmed, "/-/")
	if !found {
		project = trimmed
	}
	parts := splitPath(project)
	if len(parts) < 2 {
		return parsedSkillSource{}, fmt.Errorf("%w: gitlab URL must be group/project", ErrSkillSourceInvalid)
	}
	repo := strings.TrimSuffix(parts[len(parts)-1], ".git")
	owner := strings.Join(parts[:len(parts)-1], "/")
	src := parsedSkillSource{Kind: skillSourceGitLab, Owner: owner, Repo: repo, Ref: "HEAD"}
	if extra == "" {
		return src, nil
	}
	extraParts := splitPath(extra)
	switch extraParts[0] {
	case "tree", "blob":
		if len(extraParts) < 2 {
			return parsedSkillSource{}, fmt.Errorf("%w: gitlab tree URL is missing a ref", ErrSkillSourceInvalid)
		}
		src.Ref = extraParts[1]
		src.Subdir = strings.Join(extraParts[2:], "/")
		if extraParts[0] == "blob" && strings.EqualFold(path.Base(src.Subdir), "SKILL.md") {
			src.Subdir = path.Dir(src.Subdir)
			if src.Subdir == "." {
				src.Subdir = ""
			}
		}
	case "archive":
		src.DirectURL = u.String()
		src.Kind = skillSourceDirect
	}
	return src, nil
}

func splitPath(p string) []string {
	var out []string
	for _, part := range strings.Split(strings.Trim(p, "/"), "/") {
		if part == "" || part == "." {
			continue
		}
		out = append(out, part)
	}
	return out
}

// escapePathSegments escapes a value that spans several path segments, such as
// a "refs/heads/main" ref or a GitLab "group/subgroup" owner, without turning
// its separators into %2F.
func escapePathSegments(p string) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func isDirectArchivePath(p string) bool {
	lower := strings.ToLower(p)
	for _, ext := range []string{".zip", ".tgz", ".tar.gz", ".tar", ".md"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func (s parsedSkillSource) fetchURL() (string, error) {
	if s.DirectURL != "" {
		return s.DirectURL, nil
	}
	switch s.Kind {
	case skillSourceGitHub:
		ref := s.Ref
		if ref == "" {
			ref = "HEAD"
		}
		return fmt.Sprintf("https://codeload.github.com/%s/%s/zip/%s",
			escapePathSegments(s.Owner), url.PathEscape(s.Repo), escapePathSegments(ref)), nil
	case skillSourceGitLab:
		ref := s.Ref
		if ref == "" {
			ref = "HEAD"
		}
		return fmt.Sprintf("https://gitlab.com/%s/%s/-/archive/%s/%s.zip",
			escapePathSegments(s.Owner), url.PathEscape(s.Repo),
			escapePathSegments(ref), url.PathEscape(s.Repo+"-"+ref)), nil
	default:
		return "", fmt.Errorf("%w: cannot resolve source", ErrSkillSourceInvalid)
	}
}

// fetchedSkillSource is the archive a source resolved to, plus the skill root
// inside it.
type fetchedSkillSource struct {
	body        []byte
	contentType string
	subdir      string
}

func fetchSkillSourceBytes(
	ctx context.Context, client *http.Client, src parsedSkillSource,
) (fetchedSkillSource, error) {
	target, err := src.fetchURL()
	if err != nil {
		return fetchedSkillSource{}, err
	}
	body, contentType, err := getSkillURL(ctx, client, target)
	if err != nil {
		return fetchedSkillSource{}, err
	}
	fetched := fetchedSkillSource{body: body, contentType: contentType, subdir: src.Subdir}
	if isZipMagic(body) || looksLikeSkillMarkdown(body) || isZipPayload(contentType, body) {
		return fetched, nil
	}
	return fetchedSkillSource{}, fmt.Errorf(
		"%w: remote did not return a skill archive", ErrSkillSourceInvalid)
}

func getSkillURL(
	ctx context.Context, client *http.Client, rawURL string,
) ([]byte, string, error) {
	if err := secutils.ValidateURLForSSRF(rawURL); err != nil {
		return nil, "", fmt.Errorf("%w: %s", ErrSkillSourceInvalid,
			secutils.FormatSSRFError("skill source", rawURL, err))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("%w: invalid source URL", ErrSkillSourceInvalid)
	}
	req.Header.Set("User-Agent", skillSourceUserAgent)
	req.Header.Set("Accept", "application/zip, application/octet-stream, application/json, text/plain;q=0.9, */*;q=0.8")
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, secutils.ErrSSRFRedirectBlocked) {
			return nil, "", fmt.Errorf("%w: %s", ErrSkillSourceInvalid,
				secutils.FormatSSRFError("skill source", rawURL, err))
		}
		return nil, "", fmt.Errorf("%w: download failed: %v", ErrSkillSourceInvalid, err)
	}
	defer func() { _ = resp.Body.Close() }()

	maxBytes := secutils.GetMaxSkillBundleSize()
	// The cap is the downloaded body — a GitHub zipball is the whole
	// repository, not the SKILL.md subtree counted later at parse time.
	if resp.ContentLength > maxBytes {
		return nil, "", fmt.Errorf("%w: skill bundle cannot exceed %d MB",
			ErrSkillSourceInvalid, secutils.GetMaxSkillBundleSizeMB())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		preview, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		msg := strings.TrimSpace(string(preview))
		if msg == "" {
			msg = resp.Status
		}
		return nil, "", fmt.Errorf("%w: download returned HTTP %d: %s",
			ErrSkillSourceInvalid, resp.StatusCode, truncateSkillError(msg))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("%w: failed to read skill archive", ErrSkillSourceInvalid)
	}
	if int64(len(body)) > maxBytes {
		return nil, "", fmt.Errorf("%w: skill bundle cannot exceed %d MB",
			ErrSkillSourceInvalid, secutils.GetMaxSkillBundleSizeMB())
	}
	if len(body) == 0 {
		return nil, "", fmt.Errorf("%w: remote returned an empty body", ErrSkillSourceInvalid)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func normalizeFetchedSkill(body []byte, contentType, subdir string) (*SkillBundle, []byte, error) {
	if looksLikeSkillMarkdown(body) {
		files := map[string][]byte{"SKILL.md": body}
		bundle, err := skillBundleFromFiles(body, files)
		if err != nil {
			return nil, nil, err
		}
		archive, err := zipSkillFiles(files)
		if err != nil {
			return nil, nil, err
		}
		bundle.SHA256 = skillArchiveSHA256(archive)
		return bundle, archive, nil
	}
	if !isZipPayload(contentType, body) {
		return nil, nil, fmt.Errorf("%w: remote did not return a zip skill bundle", ErrSkillSourceInvalid)
	}
	opts := SkillBundleParseOptions{
		Subdir:           subdir,
		AllowExtraFiles:  true,
		AllowNestedSkill: true,
	}
	bundle, err := ParseSkillBundleWithOptions(body, opts)
	if err != nil {
		return nil, nil, err
	}
	archive, err := zipSkillFiles(bundle.Files)
	if err != nil {
		return nil, nil, err
	}
	bundle.SHA256 = skillArchiveSHA256(archive)
	return bundle, archive, nil
}

func isZipMagic(body []byte) bool {
	return len(body) >= 4 && bytes.HasPrefix(body, []byte("PK\x03\x04"))
}

func isZipPayload(contentType string, body []byte) bool {
	if isZipMagic(body) {
		return true
	}
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "zip")
}

func looksLikeSkillMarkdown(body []byte) bool {
	trimmed := bytes.TrimSpace(bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF}))
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return false
	}
	return bytes.Contains(trimmed, []byte("\nname:")) || bytes.Contains(trimmed, []byte("\nname :"))
}

func truncateSkillError(msg string) string {
	msg = strings.Join(strings.Fields(msg), " ")
	if len(msg) > 200 {
		return msg[:200]
	}
	return msg
}
