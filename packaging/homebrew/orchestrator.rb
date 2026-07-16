class Orchestrator < Formula
  desc "Coordinate durable full-stack product work from discovery through release"
  homepage "https://github.com/fullstack-orchestrator/fullstack-orchestrator"
  version "{{VERSION}}"
  license "Apache-2.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/fullstack-orchestrator/fullstack-orchestrator/releases/download/v{{VERSION}}/orchestrator_{{VERSION}}_Darwin_arm64.tar.gz"
      sha256 "{{DARWIN_ARM64_SHA256}}"
    else
      url "https://github.com/fullstack-orchestrator/fullstack-orchestrator/releases/download/v{{VERSION}}/orchestrator_{{VERSION}}_Darwin_x86_64.tar.gz"
      sha256 "{{DARWIN_AMD64_SHA256}}"
    end
  end

  def install
    bin.install "orchestrator"
  end

  test do
    assert_match '"status":"passed"', shell_output("#{bin}/orchestrator doctor --json")
  end
end
