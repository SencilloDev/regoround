package names

error_list := {"unallowed_name": "name %v is not allowed"}

default allow := false

allow if {
	allowed_name
	not count(errors) > 0
}

allowed_name if {
	input.name in data.rules.allowed_names
}

errors contains err if {
	not allowed_name
	err := sprintf(error_list.unallowed_name, [input.name])
}
