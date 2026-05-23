# libtorch-train

**One Capy source declaring a neural-network architecture →
a high-performance C++ training project using libtorch.**

The killer demonstration of multi-file generation for serious code:
declare a model's layers as a flat list, and Capy emits idiomatic
C++ with the right `register_module` plumbing, the forward pass, the
training loop, CMake build, and a run script.

## What you write

```
model "MNIST classifier"
    dataset MNIST
    batch_size 64
    epochs 10
    learning_rate 0.001
    optimizer adam

    layer conv2d   in 1   out 32   kernel 3
    layer relu
    layer maxpool  kernel 2
    layer conv2d   in 32  out 64   kernel 3
    layer relu
    layer maxpool  kernel 2
    layer flatten
    layer linear   in 1600 out 128
    layer relu
    layer linear   in 128  out 10
end
```

17 lines.

## What you get

```
out/
├── README.md
├── CMakeLists.txt
├── run.sh
└── src/
    ├── model.h       ← torch::nn::Module with register_module + forward()
    └── main.cpp      ← training loop with optimizer + checkpoint save
```

Generated `model.h`:

```cpp
struct MNISTClassifierImpl : torch::nn::Module {
    MNISTClassifierImpl() {
        l1 = register_module("l1", torch::nn::Conv2d(torch::nn::Conv2dOptions(1, 32, 3)));
        l4 = register_module("l4", torch::nn::Conv2d(torch::nn::Conv2dOptions(32, 64, 3)));
        l8 = register_module("l8", torch::nn::Linear(1600, 128));
        l10 = register_module("l10", torch::nn::Linear(128, 10));
    }
    torch::Tensor forward(torch::Tensor x) {
        x = l1->forward(x);
        x = torch::relu(x);
        x = torch::max_pool2d(x, 2);
        x = l4->forward(x);
        ...
        return torch::log_softmax(x, /*dim=*/1);
    }
    torch::nn::Conv2d l1{nullptr};
    torch::nn::Conv2d l4{nullptr};
    torch::nn::Linear l8{nullptr};
    torch::nn::Linear l10{nullptr};
};
```

Generated `main.cpp` has the actual training loop calling
`optimizer.zero_grad() / loss.backward() / optimizer.step()` and
saving checkpoints between epochs.

## Run

```sh
../../capy run --out-dir out lib.capy script.capy
cd out
export LIBTORCH=/path/to/libtorch
./run.sh
```

## Why this matters

ML model code has a brutal repetition pattern: every conv layer
needs to be declared as a member, registered in the constructor,
AND called in `forward()`. Three places that have to stay
synchronized. Add a layer → edit three places → recompile.

With Capy, declare a layer once. The library generates all three
places. Change the layer count, channels, or kernel size in
`script.capy`; regenerate; recompile. The C++ output is what an
expert would write by hand — there's no Capy runtime dependency.

## Beyond MNIST

The same library can target larger models — add more `layer`
lines, change the dataset, increase epochs. Capy is generating
plain libtorch C++ that runs at native speed with CUDA support
when available (the generated code already calls
`torch::cuda::is_available()`).

Targets you could swap by writing a new library (`lib_<X>.capy`):

- **PyTorch Python** (`lib_pytorch.capy`) — same source → Python
- **TensorFlow / Keras** — same architecture → tf.keras model
- **ONNX export** — generate ONNX graph definition
- **TFLite for mobile** — generate quantized inference model

Same declaration, different runtimes — the architecture is the
contract.
