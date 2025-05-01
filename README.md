# regoround

This is a rego playground that you can run locally. It also allows you to load a custom bundle into the playground to use.

## Getting Started

### Custom bundle
`regoroundctl service start --bundle-path bundle.tar.gz`

You can use the default bundle included in this repo to test with.

> [!IMPORTANT]
> Currently your policy must use the play package name.

#### Policy
```
package play

import data.names

allow := names.allow

errors := names.errors
```

####  Input
```
{
 	"name": "John"
}
```

#### Result

```
{
	"allow": true,
	"errors": []
}
```

#### Override Data
If you enter data in the data field, it will override the existing bundle data to test with.

Go ahead and place this in the data field and hit evaluate

```
{
 	"rules": {
     	"allowed_names": ["Pete"]
    }
}
```

You should now see

```
{
	"allow": false,
	"errors": [
		"name John is not allowed"
	]
}
```
