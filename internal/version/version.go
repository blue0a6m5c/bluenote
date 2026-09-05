// Package version defines BlueNote's identity independently of its upstream base.
package version

const (
	Name            = "BlueNote"
	BlueNoteVersion = "0.1.0"
	UpstreamName    = "WriteFreely"
	UpstreamVersion = "0.17.2"
	SourceURL       = "https://github.com/blue0a6m5c/bluenote"
)

// Revision is supplied by the build; source archives can build without Git.
var Revision = "unknown"

// CompatibilityVersion returns the upstream-based identification used on the network.
// Its build metadata must not be used to compare BlueNote releases.
func CompatibilityVersion() string {
	return UpstreamVersion + "+bluenote." + BlueNoteVersion
}

func CLI() string {
	return Name + " " + BlueNoteVersion + " (based on " + UpstreamName + " " + UpstreamVersion + "; commit " + Revision + ")"
}
