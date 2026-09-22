package service

import (
	"context"
	"fmt"

	"github.com/paperflow/paperflow/internal/model"
	"github.com/paperflow/paperflow/internal/repository"
)

// ---- fake 仓储 ----

type fakeUserRepo struct {
	repository.UserRepository
	users  map[uint]*model.User
	byName map[string]*model.User
	nextID uint
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[uint]*model.User{}, byName: map[string]*model.User{}}
}

func (f *fakeUserRepo) Create(ctx context.Context, u *model.User) error {
	if _, ok := f.byName[u.Username]; ok {
		return fmt.Errorf("duplicate username %s", u.Username)
	}
	f.nextID++
	u.ID = f.nextID
	f.users[u.ID] = u
	f.byName[u.Username] = u
	return nil
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id uint) (*model.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeUserRepo) FindByUsername(ctx context.Context, name string) (*model.User, error) {
	if u, ok := f.byName[name]; ok {
		return u, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeUserRepo) Update(ctx context.Context, u *model.User) error {
	f.users[u.ID] = u
	f.byName[u.Username] = u
	return nil
}

type fakePaperRepo struct {
	repository.PaperRepository
	papers map[uint]*model.Paper
	nextID uint
}

func newFakePaperRepo() *fakePaperRepo {
	return &fakePaperRepo{papers: map[uint]*model.Paper{}}
}

func (f *fakePaperRepo) Create(ctx context.Context, p *model.Paper) error {
	f.nextID++
	p.ID = f.nextID
	f.papers[p.ID] = p
	return nil
}

func (f *fakePaperRepo) Update(ctx context.Context, p *model.Paper) error {
	f.papers[p.ID] = p
	return nil
}

func (f *fakePaperRepo) FindByID(ctx context.Context, id uint) (*model.Paper, error) {
	if p, ok := f.papers[id]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakePaperRepo) FindByIDForUpdate(ctx context.Context, id uint) (*model.Paper, error) {
	if p, ok := f.papers[id]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakePaperRepo) FindByIDWithDetail(ctx context.Context, id uint) (*model.Paper, error) {
	if p, ok := f.papers[id]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakePaperRepo) List(ctx context.Context, filter model.PaperFilter, page, size int) ([]model.Paper, int64, error) {
	var items []model.Paper
	var total int64
	for _, p := range f.papers {
		if filter.Status != "" && p.Status != filter.Status {
			continue
		}
		if filter.SubmitterID > 0 && p.SubmitterID != filter.SubmitterID {
			continue
		}
		items = append(items, *p)
		total++
	}
	return items, total, nil
}

type fakeReviewRepo struct {
	repository.ReviewRepository
	reviews map[uint]*model.Review
	nextID  uint
}

func newFakeReviewRepo() *fakeReviewRepo {
	return &fakeReviewRepo{reviews: map[uint]*model.Review{}}
}

func (f *fakeReviewRepo) Create(ctx context.Context, r *model.Review) error {
	f.nextID++
	r.ID = f.nextID
	f.reviews[r.ID] = r
	return nil
}

func (f *fakeReviewRepo) Update(ctx context.Context, r *model.Review) error {
	cp := *r
	f.reviews[r.ID] = &cp
	return nil
}

func (f *fakeReviewRepo) FindByID(ctx context.Context, id uint) (*model.Review, error) {
	if r, ok := f.reviews[id]; ok {
		return r, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeReviewRepo) FindByIDForUpdate(ctx context.Context, id uint) (*model.Review, error) {
	if r, ok := f.reviews[id]; ok {
		return r, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeReviewRepo) ListByPaper(ctx context.Context, paperID uint) ([]model.Review, error) {
	var items []model.Review
	for _, r := range f.reviews {
		if r.PaperID == paperID {
			items = append(items, *r)
		}
	}
	return items, nil
}

func (f *fakeReviewRepo) FindInviteByPaperReviewer(ctx context.Context, paperID, reviewerID uint) (*model.Review, error) {
	for _, r := range f.reviews {
		if r.PaperID == paperID && r.ReviewerID == reviewerID {
			return r, nil
		}
	}
	return nil, repository.ErrNotFound
}

type fakeRevisionRepo struct {
	repository.RevisionRepository
	revisions []model.Revision
	nextID    uint
}

func newFakeRevisionRepo() *fakeRevisionRepo {
	return &fakeRevisionRepo{}
}

func (f *fakeRevisionRepo) Create(ctx context.Context, r *model.Revision) error {
	f.nextID++
	r.ID = f.nextID
	f.revisions = append(f.revisions, *r)
	return nil
}

func (f *fakeRevisionRepo) ListByPaper(ctx context.Context, paperID uint) ([]model.Revision, error) {
	var items []model.Revision
	for _, r := range f.revisions {
		if r.PaperID == paperID {
			items = append(items, r)
		}
	}
	return items, nil
}

type fakePlagiarismRepo struct {
	repository.PlagiarismRepository
	checks map[uint]*model.PlagiarismCheck
	nextID uint
}

func newFakePlagiarismRepo() *fakePlagiarismRepo {
	return &fakePlagiarismRepo{checks: map[uint]*model.PlagiarismCheck{}}
}

func (f *fakePlagiarismRepo) Create(ctx context.Context, c *model.PlagiarismCheck) error {
	f.nextID++
	c.ID = f.nextID
	f.checks[c.ID] = c
	return nil
}

func (f *fakePlagiarismRepo) Update(ctx context.Context, c *model.PlagiarismCheck) error {
	f.checks[c.ID] = c
	return nil
}

func (f *fakePlagiarismRepo) FindByPaper(ctx context.Context, paperID uint) (*model.PlagiarismCheck, error) {
	for _, c := range f.checks {
		if c.PaperID == paperID {
			return c, nil
		}
	}
	return nil, repository.ErrNotFound
}

type fakeAuditRepo struct {
	repository.AuditLogRepository
	logs []model.AuditLog
}

func newFakeAuditRepo() *fakeAuditRepo {
	return &fakeAuditRepo{}
}

func (f *fakeAuditRepo) Create(ctx context.Context, l *model.AuditLog) error {
	f.logs = append(f.logs, *l)
	return nil
}

type fakeWithdrawalRepo struct {
	repository.WithdrawalRepository
	withdrawals []*model.Withdrawal
	reviewsRef  *fakeReviewRepo
	nextID      uint
}

func newFakeWithdrawalRepo() *fakeWithdrawalRepo {
	return &fakeWithdrawalRepo{}
}

func (f *fakeWithdrawalRepo) Create(ctx context.Context, w *model.Withdrawal) error {
	for _, existing := range f.withdrawals {
		if existing.PaperID == w.PaperID && existing.Status == "pending" {
			return fmt.Errorf("duplicate pending withdrawal for paper %d", w.PaperID)
		}
	}
	f.nextID++
	w.ID = f.nextID
	f.withdrawals = append(f.withdrawals, w)
	return nil
}

func (f *fakeWithdrawalRepo) Update(ctx context.Context, w *model.Withdrawal) error {
	for i, existing := range f.withdrawals {
		if existing.ID == w.ID {
			f.withdrawals[i] = w
			return nil
		}
	}
	return repository.ErrNotFound
}

func (f *fakeWithdrawalRepo) FindByID(ctx context.Context, id uint) (*model.Withdrawal, error) {
	for _, w := range f.withdrawals {
		if w.ID == id {
			return w, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeWithdrawalRepo) FindByIDForUpdate(ctx context.Context, id uint) (*model.Withdrawal, error) {
	return f.FindByID(ctx, id)
}

func (f *fakeWithdrawalRepo) FindPendingByPaper(ctx context.Context, paperID uint) (*model.Withdrawal, error) {
	for _, w := range f.withdrawals {
		if w.PaperID == paperID && w.Status == "pending" {
			return w, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeWithdrawalRepo) FindLatestByPaper(ctx context.Context, paperID uint) (*model.Withdrawal, error) {
	var latest *model.Withdrawal
	for _, w := range f.withdrawals {
		if w.PaperID == paperID && (latest == nil || w.CreatedAt.After(latest.CreatedAt)) {
			latest = w
		}
	}
	if latest == nil {
		return nil, repository.ErrNotFound
	}
	return latest, nil
}

func (f *fakeWithdrawalRepo) List(ctx context.Context, filter model.WithdrawalFilter, page, size int) ([]model.Withdrawal, int64, error) {
	var items []model.Withdrawal
	for _, w := range f.withdrawals {
		if filter.Status != "" && w.Status != filter.Status {
			continue
		}
		if filter.PaperID > 0 && w.PaperID != filter.PaperID {
			continue
		}
		items = append(items, *w)
	}
	return items, int64(len(items)), nil
}

func (f *fakeWithdrawalRepo) CloseOpenReviews(ctx context.Context, paperID uint, closedNote string) (int64, error) {
	var n int64
	for _, r := range f.reviewsRef.reviews {
		if r.PaperID == paperID && (r.Status == "invited" || r.Status == "accepted") {
			r.Status = "closed"
			if r.Comments == "" {
				r.Comments = closedNote
			}
			n++
		}
	}
	return n, nil
}

type fakeStore struct {
	users       *fakeUserRepo
	papers      *fakePaperRepo
	reviews     *fakeReviewRepo
	revisions   *fakeRevisionRepo
	plagiarism  *fakePlagiarismRepo
	withdrawals *fakeWithdrawalRepo
	audit       *fakeAuditRepo
}

func newFakeStore() *fakeStore {
	s := &fakeStore{
		users:       newFakeUserRepo(),
		papers:      newFakePaperRepo(),
		reviews:     newFakeReviewRepo(),
		revisions:   newFakeRevisionRepo(),
		plagiarism:  newFakePlagiarismRepo(),
		withdrawals: newFakeWithdrawalRepo(),
		audit:       newFakeAuditRepo(),
	}
	s.withdrawals.reviewsRef = s.reviews
	return s
}

func (f *fakeStore) Transaction(ctx context.Context, fn func(repository.Store) error) error {
	return fn(f)
}

func (f *fakeStore) UserRepository() repository.UserRepository     { return f.users }
func (f *fakeStore) PaperRepository() repository.PaperRepository   { return f.papers }
func (f *fakeStore) ReviewRepository() repository.ReviewRepository { return f.reviews }
func (f *fakeStore) RevisionRepository() repository.RevisionRepository {
	return f.revisions
}
func (f *fakeStore) PlagiarismRepository() repository.PlagiarismRepository {
	return f.plagiarism
}
func (f *fakeStore) WithdrawalRepository() repository.WithdrawalRepository {
	return f.withdrawals
}
func (f *fakeStore) AuditLogRepository() repository.AuditLogRepository {
	return f.audit
}
