package deploy

import "strings"

func FormatProjectDescs(projects []ProjectDesc) string {
	var buf strings.Builder
	for _, pj := range projects {
		var wroteProjectHeader bool
		for _, lock := range pj.Phases {
			if !lock.Locked {
				continue
			}

			if !wroteProjectHeader {
				project := pj.Name
				buf.WriteString(project)
				buf.WriteString("\n")
				wroteProjectHeader = true
			}

			env := lock.Name
			buf.WriteString("  ")
			buf.WriteString(env)
			buf.WriteString(": ")
			buf.WriteString("Locked")
			if len(lock.LockHistory) > 0 {
				buf.WriteString(" (by ")
				buf.WriteString(lock.LockHistory[len(lock.LockHistory)-1].User)
				buf.WriteString(", for ")
				buf.WriteString(lock.LockHistory[len(lock.LockHistory)-1].Reason)
				buf.WriteString(")")
			}
			buf.WriteString("\n")
		}
	}

	return buf.String()
}
