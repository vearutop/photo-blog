:::{lang=en}
# Brief overview of Markdown syntax
:::

:::{lang=ru}
# Краткий обзор синтаксиса Markdown
:::


| Markdown                                  | HTML                                                 | Output                                  |
|-------------------------------------------|------------------------------------------------------|-----------------------------------------|
| `*italic*`                                | `<em>italic</em>`                                    | *italic*                                |
| `**bold**`                                | `<strong>bold</strong>`                              | **bold**                                |
| `***bold and italic***`                   | `<em><strong>bold and italic</strong></em>`          | ***bold and italic***                   |
| `~~wrong~~`                               |                                                      | ~~wrong~~                               |
| `text with a [link](https://example.com)` | `text with a <a href="https://example.com">link</a>` | text with a [link](https://example.com) |

:::{lang=en}
## Headings
:::

:::{lang=ru}
## Заголовки
:::

```
## Heading 2
### Heading 3
#### Heading 4
```

## Heading 2

### Heading 3

#### Heading 4

:::{lang=en}
### Image
:::

:::{lang=ru}
### Изображение
:::

```
![An icon with a camera](https://vearutop.p1cs.art/static/favicon.png)
```

![An icon with a camera](https://vearutop.p1cs.art/static/favicon.png)

:::{lang=en}
## Quote
:::

:::{lang=ru}
## Цитирование
:::

```
> blockquote
```

> blockquote

:::{lang=en}
## Table
:::

:::{lang=ru}
## Таблица
:::

```
| Col1  | Col2 |
|-------|------|
| one   | two  |
| three | four |
```

| Col1  | Col2 |
|-------|------|
| one   | two  |
| three | four |

:::{lang=en}
### List
:::

:::{lang=ru}
### Список
:::

```
* One
* Two
* Three
```

* One
* Two
* Three

:::{lang=en}
## Horizontal Rule
:::

:::{lang=ru}
## Горизонтальный разделитель
:::

```
A

---

B
```

A

---

B

:::{lang=en}
### Pixelpeep
:::

:::{lang=ru}
### Пиксель-пип
:::

:::{lang=en}
Compare 2-4 images side by side with synchronized pan/zoom, to check sharpness and detail across similarly framed shots. Wheel or pinch to zoom, drag to pan — all viewports move together, and the current zoom is shown as a percentage next to "Reset zoom". Each viewport has a lock button (top right) to nudge that one image independently, e.g. to correct for a slightly different framing; lock it again and the correction is kept while group pan/zoom resumes. Config is one JSON object: `height` sets the viewport height (omit it to fill the screen height instead), top-level `zoom`/`offsetX`/`offsetY` set the initial shared pan/zoom (also what "Reset zoom" returns to), and each entry in `items` may have its own `zoom` (to compensate for a different focal length, e.g. comparing a 35mm shot to a 50mm one) and `offsetX`/`offsetY` (to nudge that image's center by a fixed amount, for scenes that don't line up perfectly). The [pixelpeep CLI](https://github.com/vearutop/photo-blog/tree/master/cmd/pixelpeep) has a form editor (toggle with the button in the corner) to fine-tune these values interactively and copy/paste the resulting JSON.
:::

:::{lang=ru}
Сравнение 2-4 изображений рядом с синхронизированным панорамированием и масштабированием — для проверки резкости и деталей на похожих кадрах. Колесо мыши или пинч — масштаб, перетаскивание — панорамирование, все области просмотра двигаются вместе, текущий масштаб показан в процентах рядом с кнопкой "Reset zoom". У каждой области есть кнопка блокировки (справа сверху), позволяющая сдвинуть это изображение отдельно, например, чтобы скомпенсировать немного другую рамку кадра; при повторной блокировке поправка сохраняется, а групповое панорамирование/масштабирование продолжает работать. Конфигурация — единый JSON-объект: `height` задаёт высоту области просмотра (без неё область просмотра растягивается на весь экран), `zoom`/`offsetX`/`offsetY` верхнего уровня задают начальные общие панорамирование/масштаб (и то, к чему возвращает "Reset zoom"), а у каждого элемента `items` могут быть свои `zoom` (компенсация разного фокусного расстояния, например при сравнении кадров с 35мм и 50мм объективов) и `offsetX`/`offsetY` (сдвиг центра этого изображения на фиксированную величину для не идеально совпадающих кадров). В [pixelpeep CLI](https://github.com/vearutop/photo-blog/tree/master/cmd/pixelpeep) есть форма-редактор (переключается кнопкой в углу экрана) для интерактивной подгонки этих значений и копирования/вставки итогового JSON.
:::

````
:::{.pixelpeep}
```json
{
  "height": 400,
  "items": [
    {"url": "/image/hash1.jpg", "label": "35mm f/1.4"},
    {"url": "/image/hash2.jpg", "label": "50mm f/1.4", "zoom": 1.43, "offsetX": -12, "offsetY": 4}
  ]
}
```
:::
````

:::{.pixelpeep}
```json
{
  "height": 300,
  "items": [
    {"url": "https://vearutop.p1cs.art/static/favicon.png", "label": "A"},
    {"url": "https://vearutop.p1cs.art/static/favicon.png", "label": "B"}
  ]
}
```
:::
