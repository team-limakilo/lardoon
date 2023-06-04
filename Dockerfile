FROM node:lts as frontend
WORKDIR /build

ENV NODE_ENV=production

COPY src src
COPY *.json *.js ./

RUN npm ci --include=dev
RUN npm run build


FROM golang as server
WORKDIR /build

COPY cmd cmd
COPY jambon jambon
COPY *.go go.mod go.sum ./
COPY --from=frontend /build/dist dist

RUN go build cmd/lardoon/main.go


FROM debian
COPY --from=server /build/main /bin/lardoon
ENTRYPOINT ["lardoon"]
