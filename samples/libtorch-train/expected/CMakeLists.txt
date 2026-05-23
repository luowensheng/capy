cmake_minimum_required(VERSION 3.18)
project(MNISTClassifier CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(Torch REQUIRED)

add_executable(m_n_i_s_t_classifier src/main.cpp)
target_link_libraries(m_n_i_s_t_classifier "${TORCH_LIBRARIES}")
set_property(TARGET m_n_i_s_t_classifier PROPERTY CXX_STANDARD 17)
