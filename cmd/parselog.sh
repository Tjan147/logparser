#! /bin/bash

date=$1

for i in ../logs/*.log; do
    ./cmd -i $i -date $date

    cd ../analyser
    jupyter nbconvert --execute --to notebook --inplace parsed_data_plot.ipynb
    cd ../cmd
done