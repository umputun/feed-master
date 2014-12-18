# coding=utf-8

feeds = [
    ("Владимир Кара-Мурза", "http://www.echo.msk.ru/programs/graniweek/rss-audio.xml"),
    ("Народ против", "http://www.echo.msk.ru/programs/opponent/rss-audio.xml"),
    ("Ганапольское", "http://www.echo.msk.ru/programs/ganapolskoe_itogi/rss-audio.xml"),
    ("Без посредников", "http://www.echo.msk.ru/programs/nomed/rss-audio.xml"),
    ("Точка", "http://www.echo.msk.ru/programs/tochka/rss-audio.xml"),
    ("Кейс", "http://www.echo.msk.ru/programs/keys/rss-audio.xml"),
    ("Блог-аут", "http://www.echo.msk.ru/programs/blogout1/rss-audio.xml"),
    ("Альбац", "http://www.echo.msk.ru/contributors/7/rss-audio.xml"),
    ("Код доступа", "http://www.echo.msk.ru/programs/code/rss-audio.xml"),
    ("Цена Победы", "http://www.echo.msk.ru/programs/victory/rss-audio.xml"),
    ("Все так", "http://www.echo.msk.ru/programs/vsetak/rss-audio.xml"),
    ("Не так", "http://www.echo.msk.ru/programs/netak/rss-audio.xml"),
    ("В круге Света", "http://echo.msk.ru/programs/sorokina/rss-audio.xml"),
    ("Суть событий", "http://www.echo.msk.ru/programs/sut/rss-audio.xml"),
    ("Попутчики", "http://www.echo.msk.ru/programs/poputchiki/rss-audio.xml"),
    ("Русский бомбардир", "http://www.echo.msk.ru/programs/orekh_osin/rss-audio.xml"),
    ("Дилентанты", "http://echo.msk.ru/programs/Diletanti/rss-audio.xml"),
    ("Цена революции", "http://echo.msk.ru/programs/cenapobedy/rss-audio.xml"),
    ("Большой дозор", "http://echo.msk.ru/programs/dozor/rss-audio.xml"),
    ("Без дураков", "http://echo.msk.ru/programs/korzun/rss-audio.xml"),
    ("Особое мнение", "http://echo.msk.ru/programs/personalno/rss-audio.xml"),
    ("2014", "http://www.echo.msk.ru/programs/year2014/rss-audio.xml"),
    ("Интервью", "http://www.echo.msk.ru/programs/beseda/rss-audio.xml"),
    ("48 минут", "http://www.echo.msk.ru/programs/48minut/rss-audio.xml"),
    ("Выбор ясен", "http://www.echo.msk.ru/programs/vyboryasen/rss-audio.xml"),
    ("Ходорковский", "http://echo.msk.ru/guests/369/rss-audio.xml"),
]

settings = {
    "info": {
        "title": u"Эхо Москвы",
        "description": u"Правильный, комбинированный фид избранных передач (версия 2)",
        "link": "http://echo.msk.ru"
    },
    "language": "ru-ru",
    "max_items_per_feed": 5,  # how many episoded to keep per feed
    "max_items_total": 100    # total number of episodes in common feed
}
