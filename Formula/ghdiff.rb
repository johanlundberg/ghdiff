class Ghdiff < Formula
  desc "Git diff viewer in GitHub-style web UI"
  homepage "https://github.com/johanlundberg/ghdiff"
  url "https://github.com/johanlundberg/ghdiff/archive/refs/tags/v1.0.2.tar.gz"
  sha256 "bef140a1a96994029153dca8c00b1750b9a5a764fb9db2dc68d7bb40e8a29e8a"
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
    assert_match "Usage:", shell_output("#{bin}/ghdiff --help")
  end
end