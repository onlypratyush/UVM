class Uvm < Formula
  desc "Universal Version Manager for programming language runtimes"
  homepage "https://github.com/onlypratyush/UVM-"
  version "0.0.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_darwin_arm64.tar.gz"
    else
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_darwin_amd64.tar.gz"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_linux_arm64.tar.gz"
    else
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_linux_amd64.tar.gz"
    end
  end

  def install
    bin.install "uvm"
  end

  test do
    assert_match "uvm version #{version}", shell_output("#{bin}/uvm --version")
  end
end
