package service

import (
	"context"
	"testing"

	"github.com/paperflow/paperflow/internal/constants"
	"github.com/paperflow/paperflow/internal/dto"
	"github.com/paperflow/paperflow/internal/model"
)

func createWithdrawalPaper(t *testing.T, store *fakeStore, status string) *model.Paper {
	t.Helper()
	p := &model.Paper{
		Title: "撤稿流程测试论文", Abstract: "这是一篇用于撤稿单元测试的论文摘要内容，长度满足要求。",
		Keywords: "撤稿,测试", Subject: "computer", Status: status, SubmitterID: 1,
	}
	if err := store.papers.Create(context.Background(), p); err != nil {
		t.Fatalf("create paper: %v", err)
	}
	return p
}

func TestWithdrawalApplyValidation(t *testing.T) {
	store := newFakeStore()
	svc := NewWithdrawalService(store, newTestLogger())
	ctx := context.Background()

	t.Run("accepted paper cannot withdraw", func(t *testing.T) {
		p := createWithdrawalPaper(t, store, constants.PaperStatusAccepted)
		_, err := svc.Apply(ctx, 1, p.ID, dto.WithdrawalApplyRequest{Reason: "作者个人原因需要撤回投稿，特此申请。"})
		if err == nil {
			t.Fatalf("expected error for accepted paper")
		}
	})

	t.Run("not owner cannot withdraw", func(t *testing.T) {
		p := createWithdrawalPaper(t, store, constants.PaperStatusSubmitted)
		_, err := svc.Apply(ctx, 2, p.ID, dto.WithdrawalApplyRequest{Reason: "作者个人原因需要撤回投稿，特此申请。"})
		if err == nil {
			t.Fatalf("expected permission error")
		}
	})

	t.Run("external review requires alternative note", func(t *testing.T) {
		p := createWithdrawalPaper(t, store, constants.PaperStatusExternalReview)
		_, err := svc.Apply(ctx, 1, p.ID, dto.WithdrawalApplyRequest{Reason: "作者个人原因需要撤回投稿，特此申请。"})
		if err == nil {
			t.Fatalf("expected error: alternative note required")
		}
	})

	t.Run("submitted applies with reason only", func(t *testing.T) {
		p := createWithdrawalPaper(t, store, constants.PaperStatusSubmitted)
		w, err := svc.Apply(ctx, 1, p.ID, dto.WithdrawalApplyRequest{Reason: "作者个人原因需要撤回投稿，特此申请。"})
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		if w.Status != constants.WithdrawalStatusPending {
			t.Errorf("expected pending, got %s", w.Status)
		}
	})

	t.Run("duplicate pending application rejected", func(t *testing.T) {
		p := createWithdrawalPaper(t, store, constants.PaperStatusSubmitted)
		if _, err := svc.Apply(ctx, 1, p.ID, dto.WithdrawalApplyRequest{Reason: "第一次撤稿申请，原因说明足够长。"}); err != nil {
			t.Fatalf("first apply: %v", err)
		}
		if _, err := svc.Apply(ctx, 1, p.ID, dto.WithdrawalApplyRequest{Reason: "并发重复撤稿申请，原因说明足够长。"}); err == nil {
			t.Fatalf("expected duplicate application error")
		}
	})

	t.Run("revision applies with alternative note", func(t *testing.T) {
		p := createWithdrawalPaper(t, store, constants.PaperStatusRevision)
		_, err := svc.Apply(ctx, 1, p.ID, dto.WithdrawalApplyRequest{
			Reason:          "研究方案调整需要撤回修改，特此申请撤稿。",
			AlternativeNote: "后续将补充实验数据后改投其他期刊，感谢编辑部与审稿人工作。",
		})
		if err != nil {
			t.Fatalf("apply revision: %v", err)
		}
	})
}

func TestWithdrawalApproveClosesReviewsAndFreezesFlow(t *testing.T) {
	store := newFakeStore()
	wsvc := NewWithdrawalService(store, newTestLogger())
	rsvc := NewReviewService(store, newTestLogger())
	psvc := NewPaperService(store, NewPlagiarismService(store, newTestConfig(), newTestLogger()), newTestConfig(), newTestLogger())
	ctx := context.Background()

	paper := createWithdrawalPaper(t, store, constants.PaperStatusExternalReview)
	reviewer := &model.User{Username: "rev-w", Password: "x", RealName: "审稿人W", Role: constants.RoleReviewer}
	if err := store.users.Create(ctx, reviewer); err != nil {
		t.Fatalf("create reviewer: %v", err)
	}
	openReview := &model.Review{PaperID: paper.ID, ReviewerID: reviewer.ID, Status: constants.ReviewStatusAccepted}
	if err := store.reviews.Create(ctx, openReview); err != nil {
		t.Fatalf("create review: %v", err)
	}
	doneReview := &model.Review{PaperID: paper.ID, ReviewerID: reviewer.ID, Status: constants.ReviewStatusCompleted, Decision: "accept"}
	if err := store.reviews.Create(ctx, doneReview); err != nil {
		t.Fatalf("create completed review: %v", err)
	}

	w, err := wsvc.Apply(ctx, 1, paper.ID, dto.WithdrawalApplyRequest{
		Reason:          "外审阶段因数据问题申请撤稿，说明足够长。",
		AlternativeNote: "将整改后重新投稿，替代处理说明内容足够长。",
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	// 待处理期间审稿不可推进。
	if err := rsvc.Submit(ctx, openReview.ID, reviewer.ID, dto.SubmitReviewRequest{
		Decision: constants.ReviewDecisionAccept, Comments: "冻结期不应能提交评审意见内容。",
	}); err == nil {
		t.Fatalf("expected review submit blocked while withdrawal pending")
	}
	// 待处理期间终审不可推进。
	if _, err := psvc.FinalDecision(ctx, 2, paper.ID, dto.FinalDecisionRequest{
		Decision: constants.PaperStatusAccepted,
	}); err == nil {
		t.Fatalf("expected final decision blocked while withdrawal pending")
	}

	// 批准撤稿。
	w, err = wsvc.Decide(ctx, 2, w.ID, dto.WithdrawalDecisionRequest{Approve: true, Comment: "同意撤稿"})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if w.Status != constants.WithdrawalStatusApproved {
		t.Errorf("expected approved, got %s", w.Status)
	}
	p, _ := store.papers.FindByID(ctx, paper.ID)
	if p.Status != constants.PaperStatusWithdrawn {
		t.Errorf("expected paper withdrawn, got %s", p.Status)
	}
	r1, _ := store.reviews.FindByID(ctx, openReview.ID)
	if r1.Status != constants.ReviewStatusClosed {
		t.Errorf("expected open review closed, got %s", r1.Status)
	}
	r2, _ := store.reviews.FindByID(ctx, doneReview.ID)
	if r2.Status != constants.ReviewStatusCompleted {
		t.Errorf("completed review must stay unchanged, got %s", r2.Status)
	}
	// 已撤稿终态：再次申请被拒绝。
	if _, err := wsvc.Apply(ctx, 1, paper.ID, dto.WithdrawalApplyRequest{
		Reason: "终态后再次撤稿不应允许，说明足够长。",
	}); err == nil {
		t.Fatalf("expected apply blocked on withdrawn terminal state")
	}
}

func TestWithdrawalRejectRestoresFlow(t *testing.T) {
	store := newFakeStore()
	wsvc := NewWithdrawalService(store, newTestLogger())
	rsvc := NewReviewService(store, newTestLogger())
	ctx := context.Background()

	paper := createWithdrawalPaper(t, store, constants.PaperStatusInitialReview)
	reviewer := &model.User{Username: "rev-r", Password: "x", RealName: "审稿人R", Role: constants.RoleReviewer}
	if err := store.users.Create(ctx, reviewer); err != nil {
		t.Fatalf("create reviewer: %v", err)
	}
	invite := &model.Review{PaperID: paper.ID, ReviewerID: reviewer.ID, Status: constants.ReviewStatusInvited}
	if err := store.reviews.Create(ctx, invite); err != nil {
		t.Fatalf("create review: %v", err)
	}

	w, err := wsvc.Apply(ctx, 1, paper.ID, dto.WithdrawalApplyRequest{Reason: "初审阶段作者临时撤稿申请，原因说明足够长。"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	// 驳回必须填写处理意见。
	if _, err := wsvc.Decide(ctx, 2, w.ID, dto.WithdrawalDecisionRequest{Approve: false}); err == nil {
		t.Fatalf("expected comment required for reject")
	}

	if _, err := wsvc.Decide(ctx, 2, w.ID, dto.WithdrawalDecisionRequest{
		Approve: false, Comment: "撤稿理由不充分，请继续配合审稿流程。",
	}); err != nil {
		t.Fatalf("reject: %v", err)
	}
	p, _ := store.papers.FindByID(ctx, paper.ID)
	if p.Status != constants.PaperStatusInitialReview {
		t.Errorf("expected flow restored to initial_review, got %s", p.Status)
	}
	// 恢复后审稿可正常推进。
	if err := rsvc.Respond(ctx, invite.ID, reviewer.ID, true); err != nil {
		t.Fatalf("expected review respond allowed after reject: %v", err)
	}
	// 驳回后允许重新申请（一次申请语义按每篇待处理申请计）。
	if _, err := wsvc.Apply(ctx, 1, paper.ID, dto.WithdrawalApplyRequest{
		Reason:          "驳回后重新发起撤稿申请，原因说明仍然足够长。",
		AlternativeNote: "外审阶段重新申请，补充替代处理说明，内容足够长。",
	}); err != nil {
		t.Fatalf("expected re-apply allowed after rejection: %v", err)
	}
}

func TestWithdrawalDoubleDecideRejected(t *testing.T) {
	store := newFakeStore()
	wsvc := NewWithdrawalService(store, newTestLogger())
	ctx := context.Background()

	paper := createWithdrawalPaper(t, store, constants.PaperStatusSubmitted)
	w, err := wsvc.Apply(ctx, 1, paper.ID, dto.WithdrawalApplyRequest{Reason: "待重复处理测试的撤稿申请，原因足够长。"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := wsvc.Decide(ctx, 2, w.ID, dto.WithdrawalDecisionRequest{Approve: true}); err != nil {
		t.Fatalf("first decide: %v", err)
	}
	if _, err := wsvc.Decide(ctx, 2, w.ID, dto.WithdrawalDecisionRequest{Approve: false, Comment: "重复处理"}); err == nil {
		t.Fatalf("expected error on second decision")
	}
}
