GOMD2MAN = go-md2man

docs: $(patsubst %.md,%.1,$(wildcard *.md)) container-storage.conf.5

%.1: %.md
	$(GOMD2MAN) -in $^ -out $@

container-storage.conf.5: containers-storage.conf.5.md
	$(GOMD2MAN) -in $^ -out $@
