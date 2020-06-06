# Spring Model

One time-step iterates the point-masses and connect springs
Without copying, as structured as directed non-cyclic graph
and  velocities accelerated  per effecting node,  but moved
only when no other depends on its position in this step.

# WebAssembly Interface

Fun to play with.  Click  and wheel ( or arrows on bar ) to
add dots  and connect each to  existing dots  with springs,
wheeling to decide mass or spring K factor (strength).

Press the play triangle,  upper left, to run the model  and
then  wheel to select a dot  to bump  by a click, anywhere.
Observe how the connected web of point-masses shake around.

<http://rmotd.net/spring/>

# BUGS

Instability occurs.  May be due to subtle error in model or
border bounce conditions.  Maybe numeric amplifying effect.
Or maybe dampening like friction is simply required to make
things stable anyway.

Click events may lead to double-click behavior like zoom in
browser.  Should implement removal of this default handling
