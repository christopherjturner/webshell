package main

// A map of keys not to copy to the shell users env
var restrictedEnvVars = map[string]bool{
	"AUDIT_UPLOAD_URL": true,
}
