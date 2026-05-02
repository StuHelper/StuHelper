package review

func reviewPermissionRelationForAction(action string) string {
	if action == "delete" {
		return "can_delete"
	}
	return "can_hide"
}
