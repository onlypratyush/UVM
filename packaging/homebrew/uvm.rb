class Uvm < Formula
  desc "Universal Version Manager for programming language runtimes"
  homepage "https://github.com/onlypratyush/UVM-"
  version "0.0.7"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_darwin_arm64.tar.gz"
      sha256 "070e7d2560f0407e93c646fd34f6893b4a923f77bcb70bd7028355063af86fbc"
    else
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_darwin_amd64.tar.gz"
      sha256 "810a7a8f0175ffe3536876d77ef14f40f1c763aabca1e8e10a30ec475db9608f"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_linux_arm64.tar.gz"
      sha256 "e59a2efdafaa3e4a5d5033fc88db1a97697478536b9534daec3785c5a974f69b"
    else
      url "https://github.com/onlypratyush/UVM-/releases/download/v#{version}/uvm_v#{version}_linux_amd64.tar.gz"
      sha256 "d119002ec423fd53d0fe1d8d0c6ad564e7873ea9a293c3a26d74ca8d6ee7b87a"
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
