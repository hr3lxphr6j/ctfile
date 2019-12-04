package aria2

type options map[string]string

func (o *options) applyOption(opts ...option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func newOptions() *options {
	o := options(make(map[string]string, 4))
	return &o
}

type option func(*options)

func Custom(key, value string) option {
	return func(o *options) {
		(*o)[key] = value
	}
}

// The file name of the downloaded file.
// It is always relative to the directory given in --dir option.
// When the --force-sequential option is used, this option is ignored.
func Output(output string) option {
	return Custom("out", output)
}

// The directory to store the downloaded file.
func Directory(dir string) option {
	return Custom("dir", dir)
}

// Set user agent for HTTP(S) downloads. Default: aria2/$VERSION,
// $VERSION is replaced by package version.
func UserAgent(ua string) option {
	return Custom("user-agent", ua)
}
