class Uvm < Formula
  desc "Universal Version Manager for programming language runtimes"
  homepage "https://github.com/onlypratyush/UVM-"
  version "0.0.4"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_darwin_arm64.tar.gz"
      sha256 "d2af72e30b865718f7e03af6bf8987ec9cb0300df34aa0fae466d785a9c36938"
    else
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_darwin_amd64.tar.gz"
      sha256 "805e4a3ade8dde336855eb2434df9c13109b237fdb3d2c04dee43f05a9bcede2"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_linux_arm64.tar.gz"
      sha256 "52a37599bfd8ab5134a3234b9dc871619ca95002d771247b3835a2e6285dcf5b"
    else
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_linux_amd64.tar.gz"
      sha256 "6322286a57a296865d4874540033a3849cb5bfe7704256d54c86da7d2b88bcc2"
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
