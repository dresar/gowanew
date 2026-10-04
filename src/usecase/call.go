package usecase

import (
	"context"
	"fmt"
	"time"

	domainCall "github.com/dresar/gowanew/domains/call"
	"github.com/dresar/gowanew/infrastructure/whatsapp"
	pkgError "github.com/dresar/gowanew/pkg/error"
	"github.com/dresar/gowanew/pkg/utils"
	"github.com/dresar/gowanew/validations"
	"github.com/sirupsen/logrus"
)

type serviceCall struct{}

func NewCallService() domainCall.ICallUsecase {
	return &serviceCall{}
}

func (service serviceCall) RejectCall(ctx context.Context, callerJID string, callID string) error {
	if err := validations.ValidateRejectCall(ctx, callerJID, callID); err != nil {
		return err
	}

	client := whatsapp.ClientFromContext(ctx)
	if client == nil {
		return pkgError.ErrWaCLI
	}

	utils.MustLogin(client)
	parsedJID, err := utils.ParseJID(callerJID)
	if err != nil {
		return err
	}

	rejectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := client.RejectCall(rejectCtx, parsedJID, callID); err != nil {
		logrus.WithError(err).Error("Failed to reject call")
		return fmt.Errorf("failed to reject call: %w", err)
	}

	logrus.Info("Rejected call successfully")
	return nil
}
