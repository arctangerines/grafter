GOCC = go
BIN = grafter



LDFLAGS =
RELFLAGS = -ldflags="-s -w"

SRCDIR = src
BUILDDIR = build
RELEASEDIR = build/release
INSTALLDIR = ~/go/bin

SRCLIST = $(wildcard src/*.go)
SRCLIST = src/grafter.go


all : $(BUILDDIR)/$(BIN)

$(BUILDDIR)/$(BIN) : $(SRCLIST)
				@test -d $(BUILDDIR) || mkdir $(BUILDDIR)
				$(GOCC) build -o $@ $(SRCLIST)

install : release
				@test -d $(INSTALLDIR) || mkdir -p $(INSTALLDIR)
				cp $(RELEASEDIR)/$(BIN) $(INSTALLDIR)

release : $(RELEASEDIR)/$(BIN)

$(RELEASEDIR)/$(BIN) : $(SRCLIST)
				@test -d $(RELEASEDIR) || mkdir -p $(RELEASEDIR)
				$(GOCC) build -o $@ $(RELFLAGS) $(SRCLIST)


clean :
				rm -rf $(BUILDDIR)

.PHONY : all  release clean
