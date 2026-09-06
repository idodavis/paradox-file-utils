// Package modfile reads and writes the files that describe a mod on disk:
// descriptor.mod, .metadata/metadata.json, and the listing descriptions beside
// them. It also scaffolds a new mod folder.
//
// This is filesystem and packaging work, not language work. It consults the
// game registry only to learn which descriptor a game uses and where mods
// live, so it depends on game rather than living inside it.
package modfile
