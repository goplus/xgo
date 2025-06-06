ARG BASE_IMAGE=golang:1.24-bookworm

FROM $BASE_IMAGE AS build
ARG USE_GORELEASER_ARTIFACTS=0
ARG GORELEASER_ARTIFACTS_TARBALL
WORKDIR /usr/local/src/xgo
COPY . .
ENV XGOROOT=/usr/local/xgo
RUN set -eux; \
	mkdir $XGOROOT; \
	if [ $USE_GORELEASER_ARTIFACTS -eq 1 ]; then \
		tar -xzf "${GORELEASER_ARTIFACTS_TARBALL}" -C $XGOROOT; \
	else \
		git ls-tree --full-tree --name-only -r HEAD | grep -vE "^\." | xargs -I {} cp --parents {} $XGOROOT/; \
		./all.bash; \
		mv bin $XGOROOT/; \
	fi

FROM $BASE_IMAGE
ENV XGOROOT=/usr/local/xgo
COPY --from=build $XGOROOT/ $XGOROOT/
ENV PATH=$XGOROOT/bin:$PATH
WORKDIR /xgo
