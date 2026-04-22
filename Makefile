# Полный CLI (sshkmcore) на macOS 15+ со старым Go может требовать внешний линкер (LC_UUID).
# Лёгкий лаунчер sshkm собирается без cobra — только stdlib + buildinfo.
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  CORE_LDFLAGS := -ldflags=-linkmode=external
endif

.PHONY: build install test clean

# Сборка в временный файл и mv: иначе после прерванной/OOM-сборки можно оставить
# обрезанный бинарник.
build:
	go build $(CORE_LDFLAGS) -o sshkmcore.tmp ./cmd/sshkmcore
	mv -f sshkmcore.tmp sshkmcore
	go build -o sshkm.tmp ./cmd/sshkm
	mv -f sshkm.tmp sshkm
	chmod 0755 sshkm sshkmcore

install:
	go install $(CORE_LDFLAGS) ./cmd/sshkmcore
	go install ./cmd/sshkm

test:
	go test ./...

clean:
	rm -f sshkm sshkmcore sshkm.tmp sshkmcore.tmp
