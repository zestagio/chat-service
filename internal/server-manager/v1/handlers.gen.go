// Code generated by options-gen. DO NOT EDIT.
package managerv1

import (
	fmt461e464ebed9 "fmt"

	errors461e464ebed9 "github.com/kazhuravlev/options-gen/pkg/errors"
	validator461e464ebed9 "github.com/kazhuravlev/options-gen/pkg/validator"
)

type OptOptionsSetter func(o *Options)

func NewOptions(
	canReceiveProblems canReceiveProblemsUseCase,
	freeHandsSignal freeHandsSignalUseCase,
	getChats getChatsUseCase,
	getChatHistory getChatHistoryUseCase,
	resolveProblem resolveProblemUseCase,
	sendMessage sendMessageUseCase,
	options ...OptOptionsSetter,
) Options {
	o := Options{}

	// Setting defaults from field tag (if present)

	o.canReceiveProblems = canReceiveProblems

	o.freeHandsSignal = freeHandsSignal

	o.getChats = getChats

	o.getChatHistory = getChatHistory

	o.resolveProblem = resolveProblem

	o.sendMessage = sendMessage

	for _, opt := range options {
		opt(&o)
	}
	return o
}

func (o *Options) Validate() error {
	errs := new(errors461e464ebed9.ValidationErrors)
	errs.Add(errors461e464ebed9.NewValidationError("canReceiveProblems", _validate_Options_canReceiveProblems(o)))
	errs.Add(errors461e464ebed9.NewValidationError("freeHandsSignal", _validate_Options_freeHandsSignal(o)))
	errs.Add(errors461e464ebed9.NewValidationError("getChats", _validate_Options_getChats(o)))
	errs.Add(errors461e464ebed9.NewValidationError("getChatHistory", _validate_Options_getChatHistory(o)))
	errs.Add(errors461e464ebed9.NewValidationError("resolveProblem", _validate_Options_resolveProblem(o)))
	errs.Add(errors461e464ebed9.NewValidationError("sendMessage", _validate_Options_sendMessage(o)))
	return errs.AsError()
}

func _validate_Options_canReceiveProblems(o *Options) error {
	if err := validator461e464ebed9.GetValidatorFor(o).Var(o.canReceiveProblems, "required"); err != nil {
		return fmt461e464ebed9.Errorf("field `canReceiveProblems` did not pass the test: %w", err)
	}
	return nil
}

func _validate_Options_freeHandsSignal(o *Options) error {
	if err := validator461e464ebed9.GetValidatorFor(o).Var(o.freeHandsSignal, "required"); err != nil {
		return fmt461e464ebed9.Errorf("field `freeHandsSignal` did not pass the test: %w", err)
	}
	return nil
}

func _validate_Options_getChats(o *Options) error {
	if err := validator461e464ebed9.GetValidatorFor(o).Var(o.getChats, "required"); err != nil {
		return fmt461e464ebed9.Errorf("field `getChats` did not pass the test: %w", err)
	}
	return nil
}

func _validate_Options_getChatHistory(o *Options) error {
	if err := validator461e464ebed9.GetValidatorFor(o).Var(o.getChatHistory, "required"); err != nil {
		return fmt461e464ebed9.Errorf("field `getChatHistory` did not pass the test: %w", err)
	}
	return nil
}

func _validate_Options_resolveProblem(o *Options) error {
	if err := validator461e464ebed9.GetValidatorFor(o).Var(o.resolveProblem, "required"); err != nil {
		return fmt461e464ebed9.Errorf("field `resolveProblem` did not pass the test: %w", err)
	}
	return nil
}

func _validate_Options_sendMessage(o *Options) error {
	if err := validator461e464ebed9.GetValidatorFor(o).Var(o.sendMessage, "required"); err != nil {
		return fmt461e464ebed9.Errorf("field `sendMessage` did not pass the test: %w", err)
	}
	return nil
}
