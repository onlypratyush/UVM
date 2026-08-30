class Uvm < Formula
  desc "Universal Version Manager for programming language runtimes"
  homepage "https://github.com/onlypratyush/UVM-"
  version "0.0.6"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_darwin_arm64.tar.gz"
      sha256 "af20e1b6930b8bdb6f55622b2053c2c5dad8924a10e863959cd49c40a09fe50a"
    else
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_darwin_amd64.tar.gz"
      sha256 "07ad29724f3261c9fe9e4f873852dae86c449d2404a165727f247aadacd7b4d5"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_linux_arm64.tar.gz"
      sha256 "b189ff4e20ca3b6618d050db210d3f798fdcb61b3491779d113341d8ecc7adba"
    else
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_linux_amd64.tar.gz"
      sha256 "8466d17afd62ecf4ede682f008685b886618ebe181d8be43a8361d3aa89613fe"
    end
  end

  def install
    bin.install "uvm"
  end

  def caveats
    <<~EOS
      To make runtimes installed by uvm (like node, npm, etc.) available in your terminal,
      add uvm's bin directory to your PATH:

        export PATH="$HOME/.uvm/bin:$PATH"

      Add this line to your ~/.zshrc or ~/.bashrc to make it permanent.
    EOS
  end

  test do
    assert_match "uvm version #{version}", shell_output("#{bin}/uvm --version")
  end
end
