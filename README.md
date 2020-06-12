# Spring Model

A graph of connected point-masses is modeled.
Each springs ends accelerates its masses to reach the springs ideal length.
The force effected is given by a factor K and the current distance between these two points.
The connections are configured so that the graph of connected point-masses is non-cyclic.
Therefore the model implementation iterates over the masses and also moves it,
as no other distance calculation will depend on it in that same iteration.
In each surch time-step there is also a border condition that lightly bounces a mass off the frame.

# WebAssembly Interface

The web of point-masses are edited by adding a dot and scrolling the mouse-wheel,
or clicking an up/down triangle, in order to define its mass.
The mass is indicated by the area of the dot.
When a dot is added, and only at that time, it may be connected to the existing dots.
In this way a non-cyclic graph of connected dots is created,
ready for the spring-model to iterate.
When a dot is connected a line is drawn to it, representing a spring.
The K factor of the spring is adjusted by the mouse-wheel, or alternatively as described.
Clicking a second time will remove, either dot or line.

When the web drawn is satisfactory, the model may be run by clicking the right-pointing
triangle, on the top left.  Clicking anywhere will displace the last point-mass added,
which is the selected dot when first running.  Any other point-mass can be selected by
the mouse-wheel or clicking the arrow triangles.
Clicking the upper left double-rectangle will go back to edit-mode and additional dots
and lines may be added, or the existing ones removed.

<http://rmotd.net/spring/>

# BUGS

Instability sometimes occurs after a while.
It seems that both the spring model and border conditions are correct.
Therefore the probable source of instabilities is that numerical rounding
are accumulated in resonance with the springs.
Some dampening like friction in the model could be made
to overcome such amplification of numerical mishaps.
