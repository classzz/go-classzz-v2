Pod::Spec.new do |spec|
  spec.name         = 'Gczz'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/classzz/go-classzz-v2'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS Classzz Client'
  spec.source       = { :git => 'https://github.com/classzz/go-classzz-v2.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Gczz.framework'

	spec.prepare_command = <<-CMD
    curl https://gczzstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Gczz.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
