class Uvm < Formula
  desc "Universal Version Manager for programming language runtimes"
  homepage "https://github.com/onlypratyush/UVM-"
  version "0.0.5"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_darwin_arm64.tar.gz"
      sha256 "b9aa6e5e2e2f2992125dc906e650bbbfe7ae2fb3a3bad1948425ed275760665a"
    else
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_darwin_amd64.tar.gz"
      sha256 "f421b2bb3c9b44044bc12865c3e04893473e717d5c207cfbbd246a6e06b3f0f1"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_linux_arm64.tar.gz"
      sha256 "1a1f62fa9768747251f21af2f6364db1cbb306b8f8b7641a74ca5f7b92105ab6"
    else
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_linux_amd64.tar.gz"
      sha256 "d0240867d47bc38c9429649179d8f10cff92f665652f612877b7266f06b01840"
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
