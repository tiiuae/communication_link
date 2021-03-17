#!/bin/bash

build_nbr=${1:-0}
distr=${2:-focal}
arch=${3:-amd64}

get_commit() {
	echo $(git rev-parse HEAD)
}

build() {
	cd ..
	go mod download || exit
	go build || exit
	cp -f communication_link packaging/ && go clean

	cd videonode || exit
	go mod download || exit
	go build || exit
	cp -f videonode ../packaging/ && go clean
	cd ../packaging/ || exit
}

make_deb() {
	version=$1
	distribution=$2
	architecture=$3
	echo "Creating deb package..."
	build_dir=$(mktemp -d)
	mkdir "${build_dir}/DEBIAN"
	mkdir -p "${build_dir}/usr/bin/"
	cp debian/control "${build_dir}/DEBIAN/"
	cp debian/postinst "${build_dir}/DEBIAN/"
	cp debian/prerm "${build_dir}/DEBIAN/"
	cp communication_link "${build_dir}/usr/bin/"
	cp videonode "${build_dir}/usr/bin/"

	sed -i "s/VERSION/${version}/" "${build_dir}/DEBIAN/control"
	cat "${build_dir}/DEBIAN/control"

	# create changelog
	pkg_name=$(grep -oP '(?<=Package: ).*' ${build_dir}/DEBIAN/control)
	mkdir -p ${build_dir}/usr/share/doc/${pkg_name}
	cat << EOF > ${build_dir}/usr/share/doc/${pkg_name}/changelog.Debian
${pkg_name} (${version}) ${distribution}; urgency=high

  * commit: $(get_commit)

 -- $(grep -oP '(?<=Maintainer: ).*' ${build_dir}/DEBIAN/control)  $(date +'%a, %d %b %Y %H:%M:%S %z')

EOF

	gzip ${build_dir}/usr/share/doc/${pkg_name}/changelog.Debian

	debfilename=${pkg_name}_${version}_${architecture}.deb
	echo "${debfilename}"
	fakeroot dpkg-deb --build ${build_dir} ./${debfilename}
	rm -rf ${build_dir}
	echo "Done"
}

version=1.0.${build_nbr}
build
make_deb $version $distr $arch
