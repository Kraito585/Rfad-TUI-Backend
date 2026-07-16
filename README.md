Бэкенд для [Rfad-TUI-Linux-Installer](https://github.com/Kraito585/Rfad-TUI-Linux-Installer)
Написан на моём кстомном бэкенд паке [go-core-cli](https://github.com/Kraito585/go-core-cli), он создовался для разработки под k8s и максимальную оптимизацию.
На данный момент всё что он делает это перезаливает официальные обновления RFAD на моё S3 зеркало и отдаёт их через api запрос https://api.kraito.ru/api/v1/updates/latest.
Позже он будет отдовать список прессетов для Community Shader с превюшками, а также отдовать обновления кастомному лаунчеру когда я его допишу.
