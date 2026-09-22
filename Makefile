PY ?= python3
PKG := python

.PHONY: test wheel zipapp install clean

test:
	cd $(PKG) && PYTHONPATH=. $(PY) -m unittest discover -s tests

wheel:
	cd $(PKG) && $(PY) -m build

# A single-file zipapp with rich vendored in (needs Python 3.10+ to run).
zipapp:
	rm -rf build/zipapp dist
	mkdir -p build/zipapp dist
	$(PY) -m pip install --quiet --target build/zipapp rich
	cp -r $(PKG)/hyprtk_usb build/zipapp/
	printf 'import sys\nfrom hyprtk_usb.cli import main\nsys.exit(main())\n' > build/zipapp/__main__.py
	$(PY) -m zipapp build/zipapp -o dist/hyprtk-usb.pyz -p "/usr/bin/env python3"
	@echo "built dist/hyprtk-usb.pyz"

install:
	cd $(PKG) && $(PY) -m pip install --user .

clean:
	rm -rf build dist $(PKG)/dist $(PKG)/*.egg-info
	find $(PKG) -name __pycache__ -type d -prune -exec rm -rf {} +
