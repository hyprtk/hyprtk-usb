PY ?= python3
PKG := python
VENV ?= $(HOME)/.local/share/hyprtk-usb/venv
BINDIR ?= $(HOME)/.local/bin

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

# Arch's Python is PEP 668 "externally managed", so install into a dedicated venv
# with system site-packages (so it can see the system PyGObject/GTK) and expose
# the console scripts on PATH by symlink. Mirrors how hyprtk-bar installs.
install:
	$(PY) -m venv --system-site-packages $(VENV)
	$(VENV)/bin/pip install --quiet --upgrade pip
	$(VENV)/bin/pip install --quiet ./$(PKG)
	for n in hyprtk-usb hyprtk-usb-gui hyprtk-usb-helper; do \
		ln -sf $(VENV)/bin/$$n $(BINDIR)/$$n; \
	done
	install -Dm644 $(PKG)/data/hyprtk-usb.desktop $(HOME)/.local/share/applications/hyprtk-usb.desktop
	install -Dm644 $(PKG)/data/hyprtk-usb.svg $(HOME)/.local/share/icons/hicolor/scalable/apps/hyprtk-usb.svg
	-update-desktop-database $(HOME)/.local/share/applications 2>/dev/null || true
	@echo "installed into $(VENV); launchers linked into $(BINDIR)"

clean:
	rm -rf build dist $(PKG)/dist $(PKG)/*.egg-info
	find $(PKG) -name __pycache__ -type d -prune -exec rm -rf {} +
