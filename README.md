## wsconcept
порт по умолчанию 8080, настраивается в env файле

##
### endpoints:
* ws://localhost:8080/ws?device_id=00000000-0000-1111-2222-334455667788 — регистрация ws устройства
* http://localhost:8080/message — получение сообщений

### Getting Started
    docker-compose --project-name="wsconcept" up -d
