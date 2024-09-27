<h1>BURP JUNIOR</h1>

<h5>By Petr09Mitin (Петр Митин)</h5>

<h3>Как запустить</h3>
<ol>
  <li>Применить git clone</li>
  <li>Создать .env в корне проекта, вставить в него содержимое файла .env.example</li>
  <li>Сгенерировать корневой TLS-сертификат (CA), выполнив в консоли скрипт ./utils/gen_ca.sh</li>
  <li>Добавить его в доверенные сертификаты ОС</li>
  <li>Запустить команду docker-compose up --build (либо make run)</li>
  <li>Proxy ранится на порту 8080, web API - на 8000</li>
</ol>

<h3>API (:8000)</h3>
<ol>
  <li>/requests – список запросов</li>
  <li>/requests/{id} – вывод 1 запроса</li>
  <li>/requests/{id}/repeat – повторная отправка запроса</li>
  <li>/requests/{id}/scan – сканирование запроса (command injection). Возвращает только те поля запроса, которые оказались уязвимы для инъекции. https://portswigger.net/web-security/os-command-injection/lab-simple лаба для тестирования скана.</li>
</ol>
