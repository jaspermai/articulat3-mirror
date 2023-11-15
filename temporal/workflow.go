package temporal

import (
	"errors"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var (
	ErrSessionHostDown = errors.New("session host down")
)

func SessionFailureRecoveryWorkflow(ctx workflow.Context) (err error) {
	for retryNum := 0; retryNum < 3; retryNum++ {
		if err = runSession(ctx); errors.Is(err, ErrSessionHostDown) {
			if sleepErr := workflow.Sleep(ctx, 5*time.Minute); sleepErr != nil {
				return sleepErr
			}
			continue
		}
		if err != nil {
			workflow.GetLogger(ctx).Error("Workflow failed.", "Error", err.Error())
		} else {
			workflow.GetLogger(ctx).Info("Workflow completed.")
		}
		return
	}
	workflow.GetLogger(ctx).Error("Workflow failed after multiple session retries.", "Error", err.Error())
	return
}

func runSession(ctx workflow.Context) (err error) {
	so := &workflow.SessionOptions{
		CreationTimeout:  time.Minute,
		ExecutionTimeout: 20 * time.Minute,
	}

	sessionCtx, err := workflow.CreateSession(ctx, so)
	if err != nil {
		// TODO: distinguish between not being able to create a session and a host going down.
		if temporal.IsTimeoutError(err) {
			workflow.GetLogger(ctx).Error("Session failed", "Error", err.Error())
			err = ErrSessionHostDown
		}
		return err
	}

	defer func() {
		workflow.CompleteSession(sessionCtx)
		// If the session host fails any scheduled activity started on the host will be cancelled.
		if workflow.GetSessionInfo(sessionCtx).SessionState == workflow.SessionStateFailed {
			err = ErrSessionHostDown
		}
	}()

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		// When running an activity in a session you don't need to specify a heartbeat timeout to
		// detect the host going down, the session heartbeat timeout will handle that for you.
		// You may still want to specify a heartbeat timeout if the activity can get stuck or
		// you want to record progress with the heartbeat details.
		HeartbeatTimeout: 40 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
		},
	}
	sessionCtx = workflow.WithActivityOptions(sessionCtx, ao)

	var a *Activities
	defer func() {
		if !errors.Is(ctx.Err(), workflow.ErrCanceled) {
			return
		}

		newCtx, _ := workflow.NewDisconnectedContext(ctx)
		err := workflow.ExecuteActivity(newCtx, a.CleanupActivity).Get(ctx, nil)
		if err != nil {
			workflow.GetLogger(ctx).Error("CleanupActivity failed", "Error", err)
		}
	}()

	err = workflow.ExecuteActivity(sessionCtx, a.PrepareWorkerActivity).Get(sessionCtx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("Prepare failed", "Error", err)
		return err
	}

	err = workflow.ExecuteActivity(sessionCtx, a.TrainPrompt).Get(sessionCtx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("TrainPrompt failed", "Error", err)
		return err
	}

	err = workflow.ExecuteActivity(sessionCtx, a.CleanupActivity).Get(sessionCtx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("CleanupActivity failed", "Error", err)
		return err
	}

	return err
}
