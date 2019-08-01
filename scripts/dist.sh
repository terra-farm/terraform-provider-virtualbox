#/bin/sh

which sha256sum

mkdir dist
OS="linux"
case "${OSTYPE}" in
  darwin*)  OS="darwin" ;; 
  linux*)   OS="linux" ;;
  *)        echo "unknown: ${OSTYPE}" ;;
esac

cp "${GOPATH}/bin/terraform-provider-virtualbox" "dist/terraform-provider-virtualbox-${TRAVIS_TAG}-${OS}_amd64"
sha256sum "dist/terraform-provider-virtualbox-${TRAVIS_TAG}-${OS}_amd64" | awk '{ print $1 }' > "dist/terraform-provider-virtualbox-${TRAVIS_TAG}-${OS}_amd64.sha256sum"
ls -lsa dist/
cat "dist/terraform-provider-virtualbox-${TRAVIS_TAG}-${OS}_amd64.sha256sum"
