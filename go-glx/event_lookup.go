package glx

import "sort"

// FindPersonEvent finds the event of the given type where the specified person
// is a principal participant (not a witness, informant, or other role).
// When multiple matching events exist, returns the one with the lowest ID
// for deterministic behavior. Returns ("", nil) if not found.
func FindPersonEvent(archive *GLXFile, personID, eventType string) (string, *Event) {
	if archive == nil {
		return "", nil
	}

	// Sort event IDs for deterministic iteration order.
	ids := make([]string, 0, len(archive.Events))
	for id := range archive.Events {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		event := archive.Events[id]
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
