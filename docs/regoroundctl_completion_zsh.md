## regoroundctl completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(regoroundctl completion zsh)

To load completions for every new session, execute once:

#### Linux:

	regoroundctl completion zsh > "${fpath[1]}/_regoroundctl"

#### macOS:

	regoroundctl completion zsh > $(brew --prefix)/share/zsh/site-functions/_regoroundctl

You will need to start a new shell for this setup to take effect.


```
regoroundctl completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.regoround.json)
```

### SEE ALSO

* [regoroundctl completion](regoroundctl_completion.md)	 - Generate the autocompletion script for the specified shell

###### Auto generated by spf13/cobra on 12-Jun-2025
