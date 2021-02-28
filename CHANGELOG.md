# ChangeLog

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

