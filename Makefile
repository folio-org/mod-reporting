# See also: ramls/Makefile (used only for validation and documentation)

**default**: target/ModuleDescriptor.json target/mod-reporting

target/ModuleDescriptor.json:
	(cd target; make)

target/mod-reporting:
	(cd src; make)

lint:
	(cd src; make lint)

test:
	(cd src; make test)

clean:
	(cd target; make clean)

