	if delta.DifferentAt("Spec.Tags") {
		if err = syncTags(
			ctx, 
			rm.sdkapi,
			rm.metrics,
			string(*desired.ko.Status.ACKResourceMetadata.ARN),
			desired.ko.Spec.Tags,
			latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}
	if !delta.DifferentExcept("Spec.Tags") {
		return desired, nil
	}