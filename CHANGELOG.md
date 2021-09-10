# ChangeLog

## v0.1.4

- Added `AddFilepath` so users can add a hard-coded filepath to the list
  of possible config files but without adding a search path or filename.
- The function `ReadConfigNoOverwrite` was added as a way to populate the
  configuration without overwriting existing values.
- Added `RemoveFile` to remove a filename and `RemovePath` to remove a config
  search path.
- Documentation now more clearly states how `AddFile` and `AddPath` are meant to
  be used.
- Fixed `NewConfigCommand` to follow the pattern of methods on the `Config`
  struct with corresponding global functions for a global config.
- `DirUsed` deprecated in favor of `PathsUsed`
- `FileUsed` deprecated in favor of `FilesUsed`
- Can now set flag usage and shorthand when binding to a flagset using the new
  `FlagInfo` interface which can be passed to `BinfToFlagSet` and
  `BindToPFlagSet`.

## v0.1.3

- Added an explicit `ReadConfigFromFile` function for reading the config
  directly from a file.
- Fixed bug for windows build when editing with a terminal text editor.
- Added a template for cobra.Command help messages that I use in basically all
  my projects. This might be removed later as it has nothing to to with
  configuration.
- Fixed for windows support.
- Changed `AddHomeDir` to `AddUserHomeDir` and `AddConfigDir` to
  `AddUserConfigDir`.
- Fix bug in `InitDefaults` where the function would ignore all other struct
  fields if it found one struct.

## v0.1.2

- `ReadConfigFile` is deprecated for `ReadConfig` which will now handle multiple
  config files without overriding data from previously read config files.
- Added `Watch` and `Updated` for updating the config when the file changes
- The cobra command will edit with sudo if the caller does not own the config
  file.
- Deprecated `SetFilename` in favor of AddFile. This is for the future when
  multiple config files will be supported.

## v0.1.1

- Better supported multi-arg struct tags

## v0.1.0

- Added `BindToFlagSet` as a way to bind your config to a flag.FlagSet. Also
  supports the common library github.com/spf13/pflag with the `BindToPFlagSet`
  function.
- Added an exported version of setDefaults
- Changed behavior of `FileUsed` and added `Paths`. Function `FileUsed` will
  now only return a file path when it finds a file that exists. The `Paths`
  function simply returns the list of config directories stored as an
  unexported variable on the Config struct.
