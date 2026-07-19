class Stackcord < Formula
  desc "Coordinate durable full-stack product work from discovery through release"
  homepage "https://github.com/kcrmin/Stackcord"
  version "{{VERSION}}"
  license "Apache-2.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kcrmin/Stackcord/releases/download/v{{VERSION}}/stackcord_{{VERSION}}_Darwin_arm64.tar.gz"
      sha256 "{{DARWIN_ARM64_SHA256}}"
    else
      url "https://github.com/kcrmin/Stackcord/releases/download/v{{VERSION}}/stackcord_{{VERSION}}_Darwin_x86_64.tar.gz"
      sha256 "{{DARWIN_AMD64_SHA256}}"
    end
  end

  def install
    bin.install "stackcord"
  end

  test do
    assert_match '"status":"passed"', shell_output("#{bin}/stackcord doctor --json")
  end
end
