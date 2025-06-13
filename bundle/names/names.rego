package names

error_list := {"unallowed_name": "name %v is not allowed"}

user_names := input.names

default allow := false

allow if {
	allowed_names
}

allowed_names if {
	not count(unallowed_names) > 0
}

unallowed_names contains name if {
	some name in user_names
	not name in data.rules.allowed_names
}

errors contains err if {
	some name in unallowed_names
	err := sprintf(error_list.unallowed_name, [name])
}
