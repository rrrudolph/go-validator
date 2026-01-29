# GitLab → GitHub migration checklist

- [x] GitHub Actions workflow added (`.github/workflows/ci.yml`) — runs tests on push/PR to `master` or `main`.
- [ ] **Update module path** in `go.mod`: change  
  `gitlab.economicmodeling.com/rudy.selman/go-validator`  
  to  
  `github.com/YOUR_USERNAME/go-validator`  
  then run `go mod tidy`.
- [ ] Create the repo on GitHub, add it as `origin`, and push.
- [ ] (Optional) Remove `.gitlab-ci.yml` after you’re fully on GitHub.

Your Dockerfile and `docker-compose.yml` are unchanged; you can still run tests locally with  
`docker compose run test go test -v ./...`.
