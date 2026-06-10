package version

var (
	Version = "dev"
	Commit  = ""
)

func String() string {
	if Version != "" && Version != "dev" {
		return Version
	}
	if Commit != "" {
		return Commit
	}
	return "dev"
}
