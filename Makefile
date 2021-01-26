MAINS               := $(wildcard cmd/*/main.go)
CMDS                := $(patsubst cmd/%/main.go,%,$(MAINS))
BINS                := $(foreach cmd,$(CMDS),bin/$(cmd))

.PHONY: all clean

all: $(BINS)

bin/%: cmd/%/*.go
	@echo "     BUILD   $@"
	@go build -o $@ ./cmd/$(patsubst bin/%,%,$@)/*.go

clean:
	@echo "     CLEAN"
	@rm -rf bin/

print-%: ; @echo $*=$($*)
