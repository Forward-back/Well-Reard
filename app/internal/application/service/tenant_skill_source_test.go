package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gaze/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestParseSkillSource(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want parsedSkillSource
	}{
		{
			name: "github repo url",
			in:   "https://github.com/vercel-labs/agent-skills",
			want: parsedSkillSource{
				Kind: skillSourceGitHub, Owner: "vercel-labs", Repo: "agent-skills", Ref: "HEAD",
			},
		},
		{
			name: "github tree path",
			in:   "https://github.com/vercel-labs/agent-skills/tree/main/skills/web-design",
			want: parsedSkillSource{
				Kind: skillSourceGitHub, Owner: "vercel-labs", Repo: "agent-skills",
				Ref: "main", Subdir: "skills/web-design",
			},
		},
		{
			name: "github blob skille md",
			in:   "https://github.com/vercel-labs/agent-skills/blob/main/skills/web-design/SKILL.md",
			want: parsedSkillSource{
				Kind: skillSourceGitHub, Owner: "vercel-labs", Repo: "agent-skills",
				Ref: "main", Subdir: "skills/web-design",
			},
		},
		{
			name: "gitlab project",
			in:   "https://gitlab.com/group/project/-/tree/main/skills/foo",
			want: parsedSkillSource{
				Kind: skillSourceGitLab, Owner: "group", Repo: "project",
				Ref: "main", Subdir: "skills/foo",
			},
		},
		{
			name: "direct zip",
			in:   "https://example.com/skills/demo.zip",
			want: parsedSkillSource{Kind: skillSourceDirect, DirectURL: "https://example.com/skills/demo.zip"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSkillSource(tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseSkillSourceRejects(t *testing.T) {
	_, err := parseSkillSource("")
	require.ErrorIs(t, err, ErrSkillSourceInvalid)

	_, err = parseSkillSource("file:///etc/passwd")
	require.ErrorIs(t, err, ErrSkillSourceInvalid)
	require.ErrorContains(t, err, "http(s)")

	// Bare locators are refused rather than guessed.
	_, err = parseSkillSource("owner/demo")
	require.ErrorIs(t, err, ErrSkillSourceInvalid)
	require.ErrorContains(t, err, "not a URL")

	_, err = parseSkillSource("@owner/demo")
	require.ErrorIs(t, err, ErrSkillSourceInvalid)
	require.ErrorContains(t, err, "not a URL")

	_, err = parseSkillSource("my-skill")
	require.ErrorIs(t, err, ErrSkillSourceInvalid)
	require.ErrorContains(t, err, "not a URL")

	_, err = parseSkillSource("https://unknown.example.com/owner/demo")
	require.ErrorIs(t, err, ErrSkillSourceInvalid)
	require.ErrorContains(t, err, "unsupported host")
}

func TestFetchSkillArchiveRejectsAmbiguousShorthandWithoutFetching(t *testing.T) {
	_, err := fetchSkillArchive(t.Context(), "owner/demo", http.DefaultClient)
	require.ErrorIs(t, err, ErrSkillSourceInvalid)
	require.ErrorContains(t, err, "not a URL")
}

func TestFetchSkillArchiveFromDirectZip(t *testing.T) {
	archive := zipBundle(t, map[string]string{"SKILL.md": validSkillMD})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/demo.zip" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archive)
	}))
	t.Cleanup(server.Close)
	allowLoopbackSkillFetch(t)

	got, err := fetchSkillArchive(t.Context(), server.URL+"/demo.zip", server.Client())
	require.NoError(t, err)

	bundle, err := ParseSkillBundle(got)
	require.NoError(t, err)
	require.Equal(t, "pdf-tools", bundle.Name)
}

// Skill sources are read anonymously. Nothing in this flow holds a credential.
func TestFetchSkillArchiveSendsNoCredentials(t *testing.T) {
	archive := zipBundle(t, map[string]string{"SKILL.md": validSkillMD})

	var archiveAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/demo.zip" {
			archiveAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(archive)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	allowLoopbackSkillFetch(t)

	_, err := fetchSkillArchive(t.Context(), server.URL+"/demo.zip", server.Client())
	require.NoError(t, err)
	require.Empty(t, archiveAuth)
}

func TestFetchSkillArchiveFromSkillMarkdown(t *testing.T) {
	for _, prefix := range []string{"", string([]byte{0xEF, 0xBB, 0xBF})} {
		t.Run(fmt.Sprintf("prefix=%q", prefix), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/markdown")
				_, _ = w.Write([]byte(prefix + validSkillMD))
			}))
			t.Cleanup(server.Close)
			allowLoopbackSkillFetch(t)

			got, err := fetchSkillArchive(t.Context(), server.URL+"/SKILL.md", server.Client())
			require.NoError(t, err)
			bundle, err := ParseSkillBundle(got)
			require.NoError(t, err)
			require.Equal(t, "pdf-tools", bundle.Name)
			require.Equal(t, []byte(prefix+validSkillMD), bundle.Files["SKILL.md"])
		})
	}
}

func TestParseSkillBundleNestedRemoteArchive(t *testing.T) {
	data := zipBundle(t, map[string]string{
		"repo-abc/LICENSE":               "MIT",
		"repo-abc/skills/pdf/SKILL.md":   validSkillMD,
		"repo-abc/skills/pdf/extract.py": "print(1)\n",
	})
	bundle, err := ParseSkillBundleWithOptions(data, SkillBundleParseOptions{
		AllowExtraFiles:  true,
		AllowNestedSkill: true,
	})
	require.NoError(t, err)
	require.Equal(t, "pdf-tools", bundle.Name)
	require.Contains(t, bundle.Files, "extract.py")

	_, err = ParseSkillBundle(data)
	require.ErrorIs(t, err, ErrSkillBundleInvalid, "uploads stay strict about nesting")
}

func allowLoopbackSkillFetch(t *testing.T) {
	t.Helper()
	utils.SetSSRFWhitelistFromRaw("127.0.0.1,::1")
	t.Cleanup(func() { utils.SetSSRFWhitelistFromRaw("") })
}

func TestFetchSkillArchiveRejectsNonSkillHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>not a skill</html>"))
	}))
	t.Cleanup(server.Close)
	allowLoopbackSkillFetch(t)

	_, err := fetchSkillArchive(t.Context(), server.URL+"/demo.zip", server.Client())
	require.ErrorIs(t, err, ErrSkillSourceInvalid)
	require.True(t, strings.Contains(err.Error(), "skill archive") ||
		strings.Contains(err.Error(), "zip skill bundle"))
}

func TestFetchSkillArchiveRejectsOversizeBody(t *testing.T) {
	t.Setenv("MAX_FILE_SIZE_MB", "1")
	t.Setenv("MAX_SKILL_BUNDLE_SIZE_MB", "1")
	payload := strings.Repeat("z", 2<<20)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(server.Close)
	allowLoopbackSkillFetch(t)

	_, err := fetchSkillArchive(t.Context(), server.URL+"/demo.zip", server.Client())
	require.ErrorIs(t, err, ErrSkillSourceInvalid)
	require.ErrorContains(t, err, "1 MB")
}
