class Cascade < Formula
  desc "Detects which packages in a monorepo are affected by a git change"
  homepage "https://github.com/hariprakazz/cascade"
  version "0.1.1"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/hariprakazz/cascade/releases/download/v0.1.1/cascade_Darwin_arm64.tar.gz"
      sha256 "5aea22ff2caa5e43cb36317ff6795a2d270351431e4d05738a632c6087671ae3"
    end
    on_intel do
      url "https://github.com/hariprakazz/cascade/releases/download/v0.1.1/cascade_Darwin_x86_64.tar.gz"
      sha256 "86a511d6f8c98badf235fafb062714b36b64307644909a75d222812e4e5a82c8"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/hariprakazz/cascade/releases/download/v0.1.1/cascade_Linux_arm64.tar.gz"
      sha256 "e6da53bc18ea5ed46f4ab5dd5054939094bc717952199f07824b2063ec9441d1"
    end
    on_intel do
      url "https://github.com/hariprakazz/cascade/releases/download/v0.1.1/cascade_Linux_x86_64.tar.gz"
      sha256 "cc678987c1c6f1a5f26ecb769064b53c4b89f4785c33cfb9058c6532831625a8"
    end
  end

  def install
    bin.install "cascade"
  end

  test do
    system "#{bin}/cascade", "--help"
  end
end
