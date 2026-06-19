package project

import mnemonicfs "github.com/ilyachch/mnemonic/internal/fs"

// WriteMnemonicManifest writes a mnemonic.toml document through the filesystem abstraction.
func WriteMnemonicManifest(path string, manifest *MnemonicManifest) error {
	data, err := manifest.MarshalTOML()
	if err != nil {
		return err
	}

	return mnemonicfs.WriteFile(path, data, 0o644)
}
