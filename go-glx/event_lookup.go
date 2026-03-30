package glx

// FindPersonEvent finds the first event of the given type where the specified
// person is a principal participant (not a witness, informant, or other role).
// Returns the event ID and the event, or ("", nil) if not found.
func FindPersonEvent(archive *GLXFile, personID, eventType string) (string, *Event) {
	for id, event := range archive.Events {
		if event == nil || event.Type != eventType {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == personID && isSubjectRole(p.Role) {
				return id, event
			}
		}
	}
	return "", nil
}

// isSubjectRole returns true for participant roles that indicate the person
// is the subject of the event (their own birth, death, etc.) rather than
// a witness, informant, or other auxiliary role.
func isSubjectRole(role string) bool {
	switch role {
	case ParticipantRolePrincipal, "subject", "":
		return true
	default:
		return false
	}
}
