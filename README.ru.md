# Agent Ping для Cursor

Бесплатная утилита проигрывает звук на Mac, когда Cursor Agent успешно заканчивает ответ. Поддерживает macOS 13+ на Apple Silicon и Intel. Лицензия MIT.

## Установка

```sh
curl -fsSL https://raw.githubusercontent.com/sunnyby/agent-ping-cursor/main/install.sh | sh
```

Установщик скачает сборку, проверит контрольную сумму и добавит обработчик в `~/.cursor/hooks.json`. Готовая сборка работает без установки Go.

```sh
~/.local/bin/agent-ping-cursor test
~/.local/bin/agent-ping-cursor status
```

Для проверки интеграции отправьте Cursor Agent короткий запрос и дождитесь ответа. Cursor обычно подхватывает изменения автоматически. Если обработчик не появился, проверьте **Customize → Hooks** и перезапустите Cursor. [Документация Cursor](https://cursor.com/docs/hooks#troubleshooting).

## Настройка и удаление

```sh
~/.local/bin/agent-ping-cursor test --sound Glass
~/.local/bin/agent-ping-cursor install --sound Glass
~/.local/bin/agent-ping-cursor uninstall
```

`uninstall` удаляет запись утилиты и сохраняет другие настройки. Чтобы удалить и программу, после отключения выполните `rm ~/.local/bin/agent-ping-cursor`.

Звук срабатывает на локальном событии `stop` со статусом `completed`. Прерванные и ошибочные запуски проходят без сигнала. Утилита обрабатывает событие в памяти. [Описание события](https://cursor.com/docs/hooks#stop).

Параметры установки, ограничения и инструкции для сборки описаны в [README.md](README.md).
