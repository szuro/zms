$version='0.2.0'
$commit=$(git log -n 1 --pretty=format:"%H")
$time=(Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ss.fffK")


$env:GOOS="linux"
$env:GOARCH="amd64"

go build -trimpath -ldflags="-X main.Version=$version -X main.Commit=$commit -X manin.BuildDate=$time" -o zmsd .
