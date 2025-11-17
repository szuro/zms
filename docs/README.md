# ZMS Documentation Site

This directory contains the Hugo-based documentation website for ZMS (Zabbix Metric Shipper).

## Quick Start

### Prerequisites

- Hugo v0.120.0 or higher (extended version)
- Go 1.21+ (for vanity URL functionality)

### Local Development

1. Navigate to the docs directory:
   ```bash
   cd docs
   ```

2. Start the Hugo development server:
   ```bash
   hugo server -D
   ```

3. Open your browser to http://localhost:1313/zms/

### Building for Production

Build the static site:

```bash
hugo --minify
```

The generated site will be in the `public/` directory.

## Project Structure

```
docs/
├── content/              # Markdown content files
│   ├── docs/            # General documentation
│   ├── configuration/   # Configuration guides
│   ├── plugins/         # Plugin development guides
│   └── api/             # API reference
├── themes/
│   └── zms-docs/        # Custom Hugo theme
│       ├── layouts/     # HTML templates
│       └── static/      # CSS, JS, images
├── hugo.toml            # Hugo configuration
└── public/              # Generated static site (after build)
```

## Features

### Go Vanity URL

The site includes meta tags for Go vanity import paths:

```html
<meta name="go-import" content="zms.szuro.net git https://github.com/szuro/zms">
<meta name="go-source" content="...">
```

This allows users to import packages as:

```go
import "zms.szuro.net/pkg/zbx"
```

### Content Organization

- **Documentation**: General guides and getting started
- **Configuration**: Complete configuration reference
- **Plugins**: Plugin development guides and examples
- **API Reference**: Go package documentation

## Deployment

### Static Hosting

The site is designed to be hosted at `https://zms.szuro.net/`.

#### GitHub Pages

```bash
# Build the site
hugo --minify

# Deploy to GitHub Pages (example)
cd public
git init
git add .
git commit -m "Deploy documentation"
git push origin gh-pages
```

#### Nginx

```nginx
server {
    listen 80;
    server_name szuro.net;

    location /zms/ {
        alias /var/www/zms-docs/public/;
        index index.html;
        try_files $uri $uri/ =404;
    }
}
```

#### Apache

```apache
<VirtualHost *:80>
    ServerName szuro.net

    Alias /zms /var/www/zms-docs/public

    <Directory /var/www/zms-docs/public>
        Options Indexes FollowSymLinks
        AllowOverride None
        Require all granted
    </Directory>
</VirtualHost>
```

### Cloudflare Pages / Netlify / Vercel

These platforms support Hugo out of the box. Configure:

- **Build command**: `hugo --minify`
- **Publish directory**: `public`
- **Base directory**: `docs`

### Docker

Build and serve with Docker:

```dockerfile
FROM hugomods/hugo:latest as builder
COPY . /src
WORKDIR /src
RUN hugo --minify

FROM nginx:alpine
COPY --from=builder /src/public /usr/share/nginx/html/zms
```

## Updating Content

### Adding New Documentation

1. Create a new markdown file:
   ```bash
   hugo new content/docs/my-new-doc.md
   ```

2. Edit the front matter and content:
   ```yaml
   ---
   title: "My New Documentation"
   description: "Description of the page"
   weight: 10
   ---

   # Content here
   ```

3. Preview locally with `hugo server -D`

### Updating Existing Pages

Simply edit the markdown files in `content/`. Hugo will automatically rebuild on save when running `hugo server`.

## Theme Customization

The custom theme is located in `themes/zms-docs/`:

- **Layouts**: Modify HTML structure in `layouts/`
- **Styles**: Edit CSS in `static/css/style.css`
- **Configuration**: Update theme settings in `hugo.toml`

## Go Package Documentation

The site serves as both documentation and a vanity URL for Go packages. The meta tags ensure that:

```bash
go get zms.szuro.net
```

Automatically redirects to the correct GitHub repository.

## Testing Vanity URL

Test the vanity URL functionality:

```bash
# Should return HTML with go-import meta tag
curl -H "User-Agent: Go-http-client/1.1" https://zms.szuro.net/pkg/zbx
```

## Contributing

To contribute to the documentation:

1. Edit the markdown files in `content/`
2. Test locally with `hugo server`
3. Submit a pull request with your changes

## License

The documentation follows the same license as the main ZMS project.
