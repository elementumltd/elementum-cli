# Completion

Generate the autocompletion script for ei for the specified shell.
See each sub-command's help for details on how to use the generated script.

## Available Commands

| Command | Purpose |
|--|--|
| bash | Generate the autocompletion script for bash |
| fish | Generate the autocompletion script for fish |
| powershell | Generate the autocompletion script for powershell |
| zsh | Generate the autocompletion script for zsh |

## bash - TODO

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(ei completion bash)

To load completions for every new session, execute once:

#### Linux:

	ei completion bash > /etc/bash_completion.d/ei

#### macOS:

	ei completion bash > $(brew --prefix)/etc/bash_completion.d/ei

You will need to start a new shell for this setup to take effect.

```bash
# TODO disable completion descriptions
ei completion bash --no-descriptions
```

## fish - TODO

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	ei completion fish | source

To load completions for every new session, execute once:

	ei completion fish > ~/.config/fish/completions/ei.fish

You will need to start a new shell for this setup to take effect.

```bash
# TODO disable completion descriptions
ei completion fish --no-descriptions
```

## powershell - TODO

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	ei completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.

```bash
# TODO disable completion descriptions
ei completion powershell --no-descriptions
```

## zsh - TODO

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(ei completion zsh)

To load completions for every new session, execute once:

#### Linux:

	ei completion zsh > "${fpath[1]}/_ei"

#### macOS:

	ei completion zsh > $(brew --prefix)/share/zsh/site-functions/_ei

You will need to start a new shell for this setup to take effect.

```bash
# TODO disable completion descriptions
ei completion zsh --no-descriptions
```
