package plans

import "errors"

var (
	ErrPlanNotFound        = errors.New("plan not found")
	ErrPublicPlanNotFound  = errors.New("public plan not found")
	ErrTaskNotFound        = errors.New("travel task not found")
	ErrTaskNotOwned        = errors.New("travel task is owned by another user")
	ErrTaskNotSucceeded    = errors.New("travel task is not in succeeded state")
	ErrAlreadyPublished    = errors.New("plan already published")
	ErrNotPublished        = errors.New("plan is not currently published")
	ErrInvalidTitle        = errors.New("title is invalid")
	ErrInvalidVisibility   = errors.New("visibility is invalid")
	ErrPublicPlanRemoved   = errors.New("public plan is no longer available")
	ErrSourcePlanForbidden = errors.New("cannot copy unpublished plan")
)
