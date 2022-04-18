package controllers

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var predicateLogger = log.Log.WithName("predicate").WithName("eventFilters")

type LabeledForDynamicConfigurationPredicate struct {
	predicate.Funcs
}

func (LabeledForDynamicConfigurationPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectNew == nil {
		predicateLogger.Error(nil, "Update event has no new object for update", "event", e)
		return false
	}

	if val, ok := e.ObjectNew.GetLabels()[dynamicConfigurationLabelKey]; ok {
		return val == dynamicConfigurationLabelValueWatch
	}

	predicateLogger.Info("Missing configuration-watch label, ignoring", "object", e.ObjectNew.GetName())
	return false
}
