package service

import (
	"testing"

	"gaze/internal/types"
)

func TestHasApprovalFromOther(t *testing.T) {
	submitter := "user-submitter"
	other := "user-reviewer"
	third := "user-auditor"

	cases := []struct {
		name      string
		reviewers []*types.DocCRReviewer
		want      bool
	}{
		{
			name:      "no reviewers",
			reviewers: nil,
			want:      false,
		},
		{
			name: "pending only",
			reviewers: []*types.DocCRReviewer{
				{UserID: other, Status: types.DocCRReviewerPending},
			},
			want: false,
		},
		{
			name: "submitter approved only",
			reviewers: []*types.DocCRReviewer{
				{UserID: submitter, Status: types.DocCRReviewerApproved},
			},
			want: false,
		},
		{
			name: "other approved",
			reviewers: []*types.DocCRReviewer{
				{UserID: other, Status: types.DocCRReviewerApproved},
			},
			want: true,
		},
		{
			name: "auditor approved without assignment",
			reviewers: []*types.DocCRReviewer{
				{UserID: third, Status: types.DocCRReviewerApproved},
			},
			want: true,
		},
		{
			name: "changes requested does not count",
			reviewers: []*types.DocCRReviewer{
				{UserID: other, Status: types.DocCRReviewerChangesRequested},
			},
			want: false,
		},
		{
			name: "submitter plus other approved",
			reviewers: []*types.DocCRReviewer{
				{UserID: submitter, Status: types.DocCRReviewerApproved},
				{UserID: other, Status: types.DocCRReviewerApproved},
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasApprovalFromOther(tc.reviewers, submitter); got != tc.want {
				t.Fatalf("hasApprovalFromOther() = %v, want %v", got, tc.want)
			}
		})
	}
}
