package pagetitle

import "testing"

func TestTitle(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "og:title",
			html: `<html><head><meta property="og:title" content="Моё видео"></head><body><h1>Другое</h1></body></html>`,
			want: "Моё видео",
		},
		{
			name: "og:title, атрибуты в другом порядке",
			html: `<head><meta content="Видео с content впереди" property="og:title"></head>`,
			want: "Видео с content впереди",
		},
		{
			name: "twitter:title без og",
			html: `<head><meta name="twitter:title" content="Твит-заголовок"></head>`,
			want: "Твит-заголовок",
		},
		{
			name: "h1 без meta",
			html: `<body><h1 class="video"><span>Заголовок из H1</span></h1></body>`,
			want: "Заголовок из H1",
		},
		{
			name: "title без meta и h1",
			html: `<head><title>Просто Тайтл &amp; ещё</title></head>`,
			want: "Просто Тайтл & ещё",
		},
		{
			name: "og:title приоритетнее h1",
			html: `<head><meta property="og:title" content="OG"></head><body><h1>H1</h1></body>`,
			want: "OG",
		},
		{
			name: "пусто",
			html: `<html><body><p>нет заголовка</p></body></html>`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Title(tt.html); got != tt.want {
				t.Errorf("Title = %q, want %q", got, tt.want)
			}
		})
	}
}
