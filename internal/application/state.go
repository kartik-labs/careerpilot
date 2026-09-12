package application

// validTransitions enumerates the only allowed Status -> Status edges,
// combining the preparation flow (docs/IMPLEMENTATION-PLAN.md section 4)
// with the browser state machine (CLAUDE.md "Browser State Machine",
// section 9 of the original architecture spec). CANCELLED is reachable
// from any pre-submission state; FAILED/EXPIRED/HUMAN_REQUIRED are
// exceptional states reachable from the in-flight browser stages.
var validTransitions = map[Status][]Status{
	StatusDiscovered: {StatusQualified, StatusCancelled},
	StatusQualified:  {StatusPreparing, StatusCancelled},
	StatusPreparing:  {StatusReview, StatusCancelled},
	StatusReview:     {StatusReady, StatusPreparing, StatusCancelled},
	StatusReady:      {StatusStarting, StatusCancelled},
	StatusStarting:   {StatusNavigating, StatusFailed, StatusCancelled},
	StatusNavigating: {StatusFilling, StatusHumanReq, StatusFailed},
	StatusFilling:    {StatusVerifying, StatusHumanReq, StatusFailed},
	StatusVerifying:  {StatusSubmitting, StatusHumanReq, StatusFailed},
	StatusSubmitting: {StatusSubmitted, StatusFailed},
	StatusHumanReq:   {StatusNavigating, StatusFilling, StatusVerifying, StatusExpired, StatusCancelled},
	StatusSubmitted:  {},
	StatusFailed:     {},
	StatusExpired:    {},
	StatusCancelled:  {},
}

// ValidTransition reports whether moving from `from` to `to` is allowed.
func ValidTransition(from, to Status) bool {
	for _, allowed := range validTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// Transition moves an Application to a new status if the transition is
// valid, returning an error otherwise. This is the only way Status should
// change — never assign a.Status directly.
func Transition(a Application, to Status) (Application, error) {
	if !ValidTransition(a.Status, to) {
		return Application{}, &InvalidTransitionError{From: a.Status, To: to}
	}
	a.Status = to
	return a, nil
}

// InvalidTransitionError reports an attempted illegal state transition.
type InvalidTransitionError struct {
	From, To Status
}

func (e *InvalidTransitionError) Error() string {
	return "application: invalid transition from " + string(e.From) + " to " + string(e.To)
}
