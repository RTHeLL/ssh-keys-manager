# sshkm

Профессиональный CLI-менеджер SSH-ключей для macOS, Debian и Ubuntu.

`sshkm` хранит управляемые ключи в изолированной директории `~/.ssh/sshkm`, генерирует/импортирует пары ключей, выводит fingerprint и интегрируется с `ssh-agent`.

Сборка состоит из **двух** исполняемых файлов: лёгкий **`sshkm`** (лаунчер, `version` без Cobra) и **`sshkmcore`** с полным CLI. Команда `sshkm …` подменяет процесс на `sshkmcore` (рядом в том же каталоге или в `PATH`).

## Возможности

- Безопасное хранение ключей с корректными правами (`0700` для каталога, `0600`/`0644` для ключей)
- Генерация ключей (`ed25519`, `rsa`)
- Импорт существующих ключей
- Просмотр публичного ключа и fingerprint
- Добавление/удаление ключа из `ssh-agent`
- Метаданные ключа: назначение, владелец, проект, теги, заметки
- Интеллектуальный поиск дублей: одноименные ключи и одинаковый key material
- Автоматическая сборка релизов для Linux/macOS

## Установка

### Homebrew (macOS/Linux)

После публикации релиза:

```bash
brew tap RTHeLL/homebrew-tap
brew install sshkm
```

### Debian/Ubuntu (APT/.deb)

После публикации релиза:

```bash
sudo apt install ./sshkm_<version>_linux_amd64.deb
```

### Ручная установка

Скачайте архив из Releases, распакуйте и переместите **оба** файла в каталог из `PATH`:

```bash
install -m 0755 sshkm sshkmcore /usr/local/bin/
```

## Быстрый старт

```bash
sshkm init
sshkm generate work --comment "work@laptop"
sshkm annotate work --purpose "GitHub deploy key" --project "infra" --owner "devops" --tags "prod,github"
sshkm list
sshkm list --details
sshkm info work
sshkm discover
sshkm public work
sshkm fingerprint work
sshkm agent add work
```

## Команды

- `sshkm init` — инициализировать каталог менеджера
- `sshkm list` — показать управляемые ключи
- `sshkm list --details` — показать расширенную информацию (алгоритм, purpose, tags, fingerprint)
- `sshkm info <name>` — подробная карточка ключа
- `sshkm annotate <name> --purpose ... --project ... --owner ... --tags ... --notes ...` — добавить бизнес-контекст
- `sshkm discover` — просканировать SSH-ключи и показать дубли имен/материала
- `sshkm generate <name>` — сгенерировать ключ
- `sshkm import <name> --from <private-key-path>` — импортировать ключ
- `sshkm public <name>` — вывести публичный ключ
- `sshkm fingerprint <name>` — вывести fingerprint
- `sshkm agent add <name>` — добавить ключ в `ssh-agent`
- `sshkm agent remove <name>` — удалить ключ из `ssh-agent`
- `sshkm delete <name>` — удалить ключ из менеджера
- `sshkm version` — версия сборки

## Публикация в пакетные менеджеры

Проект использует `GoReleaser`:

- `Homebrew` formula публикуется в `RTHeLL/homebrew-tap`
- `.deb` пакет публикуется как артефакт релиза (для Debian/Ubuntu)

Триггер публикации:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Перед запуском релиза добавьте GitHub Actions secret:

- `HOMEBREW_TAP_GITHUB_TOKEN` — токен с доступом на запись в tap-репозиторий.

## Локальная разработка

```bash
go test ./...
go run ./cmd/sshkmcore --help
make build
./sshkm version
./sshkm list
```

Установка в `GOPATH/bin` без `Makefile`:

```bash
go install ./cmd/sshkmcore
go install ./cmd/sshkm
```

### macOS 15+ (Sequoia и новее): `dyld: missing LC_UUID load command`

У **полного** CLI (`sshkmcore`) на старом Go без `LC_UUID` macOS может падать с `abort trap`. В `make build` для Darwin к `sshkmcore` добавляется `-linkmode=external`; лаунчер `sshkm` собирается обычным способом и обычно запускается без этой проблемы.

Долгосрочно имеет смысл обновить Go с https://go.dev/dl/ до актуальной ветки (1.24+), где поведение линкера согласовано с требованиями ОС.

### Команда «зависает» (в т.ч. `./sshkm version`), Ctrl+C не помогает

Частые причины:

1. **Обрезанный бинарник** — если сборку прервали или убил OOM (например, `exit code 137`), в каталоге мог остаться **неполный** `sshkm` или `sshkmcore`. Удалите оба файла и пересоберите: `make build` пишет во временные `*.tmp`, затем атомарно переименовывает.

2. **Слишком старый Go на очень новой macOS** — обновите toolchain (1.24+); после этого `sshkmcore` часто можно собирать без `linkmode=external`.

3. **Собран только лаунчер** — если выполнить `go build -o sshkm ./cmd/sshkm` без **`sshkmcore`** рядом (или в `PATH`), команды кроме `version` не запустятся. Используйте `make build` или соберите оба бинарника.

Диагностика зависания:

```bash
file ./sshkm
ls -la ./sshkm
# в другом терминале, пока «висит»:
sample $(pgrep -n sshkm) 1 -file /tmp/sshkm-sample.txt
```

Или отправьте процессу **SIGQUIT** (часто печатает стек рантайма): `kill -QUIT $(pgrep -n sshkm)`.
