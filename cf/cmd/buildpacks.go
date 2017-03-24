package cmd

var Buildpacks = map[string]string{
	"binary_buildpack":      "https://github.com/cloudfoundry/binary-buildpack",
	"dotnet_core_buildpack": "https://github.com/cloudfoundry/dotnet-core-buildpack",
	"go_buildpack":          "https://github.com/cloudfoundry/go-buildpack",
	"java_buildpack":        "https://github.com/cloudfoundry/java-buildpack",
	"nodejs_buildpack":      "https://github.com/cloudfoundry/nodejs-buildpack",
	"php_buildpack":         "https://github.com/cloudfoundry/php-buildpack",
	"python_buildpack":      "https://github.com/cloudfoundry/python-buildpack",
	"ruby_buildpack":        "https://github.com/cloudfoundry/ruby-buildpack",
	"staticfile_buildpack":  "https://github.com/cloudfoundry/staticfile-buildpack",
}

var BuildpackOrder = []string{
	"staticfile_buildpack",
	"java_buildpack",
	"ruby_buildpack",
	"nodejs_buildpack",
	"go_buildpack",
	"python_buildpack",
	"php_buildpack",
	"dotnet_core_buildpack",
	"binary_buildpack",
}
