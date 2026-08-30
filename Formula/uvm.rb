class Uvm < Formula
  desc "Universal Version Manager for programming language runtimes"
  homepage "https://github.com/onlypratyush/UVM-"
  version "0.0.6"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_darwin_arm64.tar.gz"
      sha256 "5984435ad5e01e9f6e2f5e786bb9760f9871081758f54dab23c626ffcadb50ed"
    else
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_darwin_amd64.tar.gz"
      sha256 "58201c0c025f2be93ed2d56ed6dc005eacc62bdb1006e56726d74da1e9e106cf"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_linux_arm64.tar.gz"
      sha256 "d20fcfc80a8f020cb4cad75ba4c3f4b6c9c19489de34bc964d4c9ca1ddc0cc73"
    else
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_linux_amd64.tar.gz"
      sha256 "3fb94f7fda6de4702518b46b132864de8fce80c9ea5d1697374d1156298d0ad6"
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
