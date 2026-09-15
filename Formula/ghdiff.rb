class Ghdiff < Formula
  desc "Git diff viewer in GitHub-style web UI"
  homepage "https://github.com/johanlundberg/ghdiff"
  url "https://github.com/johanlundberg/ghdiff/archive/refs/tags/v1.0.3.tar.gz"
  sha256 "9ff2cc0fa5e046babfd0067e33f813e46d22af2af91eb731df51b3d65d7940e3"
  license "BSD-2-Clause"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X main.version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags)

    man1.install "man/ghdiff.1"
    bash_completion.install "completions/ghdiff.bash"
    zsh_completion.install "completions/_ghdiff"
    fish_completion.install "completions/ghdiff.fish"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/ghdiff --version")
    assert_match "Usage:", shell_output("#{bin}/ghdiff --help 2>&1")
  end
end
